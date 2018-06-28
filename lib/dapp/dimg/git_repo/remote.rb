module Dapp
  module Dimg
    module GitRepo
      class Remote < Base
        CACHE_VERSION = 2

        attr_reader :url

        class << self
          def get_or_create(dapp, name, url:, ignore_git_fetch: false)
            key         = [url, ignore_git_fetch]
            inverse_key = [url, !ignore_git_fetch]

            repositories[key] ||= begin
              if repositories.key?(inverse_key)
                repositories[inverse_key]
              else
                new(dapp, name, url: url)
              end.tap do |repo|
                repo.fetch! unless ignore_git_fetch
              end
            end
          end

          def repositories
            @repositories ||= {}
          end
        end

        def initialize(dapp, name, url:)
          super(dapp, name)

          @url = url

          _with_lock do
            dapp.log_secondary_process(dapp.t(code: 'process.git_artifact_clone', data: { url: url }), short: true) do
              begin
                if [:https, :ssh].include?(remote_origin_url_protocol) && !Rugged.features.include?(remote_origin_url_protocol)
                  raise Error::Rugged, code: :rugged_protocol_not_supported, data: { url: url, protocol: remote_origin_url_protocol }
                end

                Rugged::Repository.clone_at(url, path.to_s, bare: true, credentials: _rugged_credentials)
              rescue Rugged::NetworkError, Rugged::SslError, Rugged::OSError => e
                raise Error::Rugged, code: :rugged_remote_error, data: { message: e.message, url: url }
              end
            end
          end unless path.directory?
        end

        def _with_lock(&blk)
          dapp.lock("remote_git_artifact.#{name}", default_timeout: 600, &blk)
        end

        def _rugged_credentials
          @_rugged_credentials ||= begin
            if remote_origin_url_protocol == :ssh
              host_with_user = url.split(':', 2).first
              username = host_with_user.split('@', 2).reverse.last
              Rugged::Credentials::SshKeyFromAgent.new(username: username)
            end
          end
        end

        def path
          Pathname(dapp.build_path("remote_git_repo", CACHE_VERSION.to_s, dapp.consistent_uniq_slugify(name)).to_s)
        end

        def fetch!
          _with_lock do
            cfg_path = path.join("config")
            cfg = IniFile.load(cfg_path)
            remote_origin_cfg = cfg['remote "origin"']

            old_url = remote_origin_cfg["url"]
            if old_url and old_url != url
              remote_origin_cfg["url"] = url
              cfg.write(filename: cfg_path)
            end

            dapp.log_secondary_process(dapp.t(code: 'process.git_artifact_fetch', data: { url: url }), short: true) do
              begin
                git.remotes.each { |remote| remote.fetch(credentials: _rugged_credentials) }
              rescue Rugged::SshError, Rugged::NetworkError => e
                raise Error::Rugged, code: :rugged_remote_error, data: { url: url, message: e.message.strip }
              end
            end
          end unless dapp.dry_run?
        end

        def latest_commit(branch)
          git.ref("refs/remotes/#{branch_format(branch)}").tap do |ref|
            raise Error::Rugged, code: :branch_not_exist_in_remote_git_repository, data: { branch: branch, url: url } if ref.nil?
            break ref.target_id
          end
        end

        def lookup_commit(commit)
          super
        rescue Rugged::OdbError, TypeError => _e
          raise Error::Rugged, code: :commit_not_found_in_remote_git_repository, data: { commit: commit, url: url }
        end

        def submodules_git(commit)
          submodules_git_path(commit).tap do |git_path|
            break begin
              if git_path.directory?
                Rugged::Repository.new(git_path.to_s)
              else
                Rugged::Repository.clone_at(path.to_s, git_path.to_s).tap do |submodules_git|
                  submodules_git.checkout(commit)
                end
              end
            end
          end
        end

        def submodules_git_path(commit)
          Pathname(File.join(dapp.host_docker_tmp_config_dir, "submodule", dapp.consistent_uniq_slugify(name), commit).to_s)
        end

        protected

        def git
          super(bare: true, credentials: _rugged_credentials)
        end

        private

        def branch_format(name)
          "origin/#{name.reverse.chomp('origin/'.reverse).reverse}"
        end
      end
    end
  end
end
