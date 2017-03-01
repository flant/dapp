module Dapp
  # Dimg
  class Dimg
    include GitArtifact
    include Path
    include Tags
    include Stages

    include Helper::Sha256

    attr_reader :config
    attr_reader :ignore_git_fetch
    attr_reader :should_be_built
    attr_reader :dapp

    def initialize(config:, dapp:, should_be_built: false, ignore_git_fetch: false)
      @config = config
      @dapp = dapp

      @ignore_git_fetch = ignore_git_fetch
      @should_be_built = should_be_built

      raise Error::Dimg, code: :dimg_not_built if should_be_built?
    end

    def build!
      with_introspection do
        dapp.lock("#{dapp.name}.images", readonly: true) do
          last_stage.build_lock! do
            begin
              builder.before_build_check
              last_stage.build!
            ensure
              after_stages_build!
            end
          end
        end
      end
    ensure
      cleanup_tmp
    end

    def after_stages_build!
      return unless last_stage.image.built? || dev_mode?
      last_stage.save_in_cache!
      artifacts.each { |artifact| artifact.last_stage.save_in_cache! }
    end

    def tag!(tag)
      dapp.lock("#{dapp.name}.images", readonly: true) do
        dimg_name = config._name
        if dapp.dry_run?
          dapp.log_state(dimg_name, state: dapp.t(code: 'state.tag'), styles: { status: :success })
        else
          dapp.log_process(dimg_name, process: dapp.t(code: 'status.process.tagging')) do
            last_stage.image.tag!(tag)
          end
        end
      end
    end

    def export!(repo, format:)
      dapp.lock("#{dapp.name}.images", readonly: true) do
        tags.each do |tag|
          image_name = format % { repo: repo, dimg_name: config._name, tag: tag }
          export_base!(last_stage.image, image_name)
        end
      end
    end

    def export_stages!(repo, format:)
      dapp.lock("#{dapp.name}.images", readonly: true) do
        export_images.each do |image|
          image_name = format % { repo: repo, signature: image.name.split(':').last }
          export_base!(image, image_name)
        end
      end
    end

    def export_base!(image, image_name)
      if dapp.dry_run?
        dapp.log_state(image_name, state: dapp.t(code: 'state.push'), styles: { status: :success })
      else
        dapp.lock("image.#{hashsum image_name}") do
          ::Dapp::Image::Stage.cache_reset(image_name)
          dapp.log_process(image_name, process: dapp.t(code: 'status.process.pushing')) do
            dapp.with_log_indent do
              image.export!(image_name)
            end
          end
        end
      end
    end

    def import_stages!(repo, format:)
      dapp.lock("#{dapp.name}.images", readonly: true) do
        import_images.each do |image|
          begin
            image_name = format % { repo: repo, signature: image.name.split(':').last }
            import_base!(image, image_name)
          rescue Error::Shellout
            next
          end
          break unless dapp.pull_all_stages?
        end
      end
    end

    def import_base!(image, image_name)
      if dapp.dry_run?
        dapp.log_state(image_name, state: dapp.t(code: 'state.pull'), styles: { status: :success })
      else
        dapp.lock("image.#{hashsum image_name}") do
          dapp.log_process(image_name,
                              process: dapp.t(code: 'status.process.pulling'),
                              status: { failed: dapp.t(code: 'status.failed.not_pulled') },
                              style: { failed: :secondary }) do
            image.import!(image_name)
          end
        end
      end
    end

    def run(docker_options, command)
      cmd = "docker run #{[docker_options, last_stage.image.built_id, command].flatten.compact.join(' ')}"
      if dapp.dry_run?
        dapp.log(cmd)
      else
        system(cmd) || raise(Error::Dimg, code: :dimg_not_run)
      end
    end

    def stage_image_name(stage_name)
      stages.find { |stage| stage.send(:name) == stage_name }.image.name
    end

    def builder
      @builder ||= Builder.const_get(config._builder.capitalize).new(self)
    end

    def artifacts
      @artifacts ||= artifacts_stages.map { |stage| stage.artifacts.map { |artifact| artifact[:dimg] } }.flatten
    end

    def artifact?
      false
    end

    def scratch?
      config._docker._from.nil?
    end

    def dev_mode?
      config._dev_mode || dapp.dev_mode?
    end

    def build_cache_version
      [::Dapp::BUILD_CACHE_VERSION, dev_mode? ? 1 : 0]
    end

    def introspect_image!(image:, options:)
      cmd = "docker run -ti --rm --entrypoint #{dapp.bash_bin} #{options} #{image}"
      system(cmd)
    end

    def cleanup_tmp
      FileUtils.rm_rf(tmp_path)
      artifacts.each(&:cleanup_tmp)
    end

    def stage_should_be_introspected?(name)
      dapp.cli_options[:introspect_stage] == name
    end

    protected

    def should_be_built?
      should_be_built && begin
        builder.before_dimg_should_be_built_check
        !last_stage.image.tagged?
      end
    end

    def with_introspection
      yield
    rescue Exception::IntrospectImage => e
      data = e.net_status[:data]
      introspect_image!(image: data[:built_id], options: data[:options])
      raise data[:error]
    end
  end # Dimg
end # Dapp
