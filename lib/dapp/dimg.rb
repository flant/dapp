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
    attr_reader :project

    def initialize(config:, project:, should_be_built: false, ignore_git_fetch: false)
      @config = config
      @project = project

      @ignore_git_fetch = ignore_git_fetch
      @should_be_built = should_be_built

      raise Error::Dimg, code: :dimg_not_built if should_be_built?
    end

    def build!
      project.lock("#{project.name}.images", readonly: true) do
        last_stage.build_lock! do
          begin
            builder.before_build_check
            last_stage.build!
          ensure
            last_stage.save_in_cache! if last_stage.image.built? || dev_mode?
          end
        end
      end
    rescue Error::ImageBuildFailed => e
      if project.introspect_error? || project.introspect_before_error?
        data = e.net_status[:data]
        introspect_image!(image: data[:built_id], options: data[:options])
      end

      raise
    ensure
      cleanup_tmp
    end

    def tag!(tag)
      project.lock("#{project.name}.images", readonly: true) do
        dimg_name = config._name
        if project.dry_run?
          project.log_state(dimg_name, state: project.t(code: 'state.tag'), styles: { status: :success })
        else
          project.log_process(dimg_name, process: project.t(code: 'status.process.tagging')) do
            last_stage.image.tag!(tag)
          end
        end
      end
    end

    def export!(repo, format:)
      project.lock("#{project.name}.images", readonly: true) do
        tags.each do |tag|
          image_name = format % { repo: repo, dimg_name: config._name, tag: tag }
          export_base!(last_stage.image, image_name)
        end
      end
    end

    def export_stages!(repo, format:)
      project.lock("#{project.name}.images", readonly: true) do
        export_images.each do |image|
          image_name = format % { repo: repo, signature: image.name.split(':').last }
          export_base!(image, image_name)
        end
      end
    end

    def export_base!(image, image_name)
      if project.dry_run?
        project.log_state(image_name, state: project.t(code: 'state.push'), styles: { status: :success })
      else
        project.lock("image.#{hashsum image_name}") do
          Dapp::Image::Stage.cache_reset(image_name)
          project.log_process(image_name, process: project.t(code: 'status.process.pushing')) do
            project.with_log_indent do
              image.export!(image_name)
            end
          end
        end
      end
    end

    def import_stages!(repo, format:)
      project.lock("#{project.name}.images", readonly: true) do
        import_images.each do |image|
          begin
            image_name = format % { repo: repo, signature: image.name.split(':').last }
            import_base!(image, image_name)
          rescue Error::Shellout
            next
          end
          break unless project.pull_all_stages?
        end
      end
    end

    def import_base!(image, image_name)
      if project.dry_run?
        project.log_state(image_name, state: project.t(code: 'state.pull'), styles: { status: :success })
      else
        project.lock("image.#{hashsum image_name}") do
          project.log_process(image_name,
                              process: project.t(code: 'status.process.pulling'),
                              status: { failed: project.t(code: 'status.failed.not_pulled') },
                              style: { failed: :secondary }) do
            image.import!(image_name)
          end
        end
      end
    end

    def run(docker_options, command)
      cmd = "docker run #{[docker_options, last_stage.image.name, command].flatten.compact.join(' ')}"
      if project.dry_run?
        project.log(cmd)
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
      config._dev_mode || project.dev_mode?
    end

    def build_cache_version
      [Dapp::BUILD_CACHE_VERSION, dev_mode? ? 1 : 0]
    end

    def introspect_image!(image:, options:)
      cmd = "docker run -ti --rm --entrypoint #{project.bash_bin} #{options} #{image}"
      system(cmd)
    end

    def cleanup_tmp
      # В tmp-директории могли остаться файлы, владельцами которых мы не являемся.
      # Такие файлы могут попасть туда при экспорте файлов артефакта.
      # Чтобы от них избавиться — запускаем docker-контейнер под root-пользователем
      # и удаляем примонтированную tmp-директорию.
      cmd = "".tap do |cmd|
        cmd << "docker run --rm"
        cmd << " --volume #{tmp_base_dir}:#{tmp_base_dir}"
        cmd << " ubuntu:16.04"
        cmd << " rm -rf #{tmp_path}"
      end
      project.shellout! cmd

      artifacts.each(&:cleanup_tmp)
    end

    protected

    def should_be_built?
      should_be_built && begin
        builder.before_dimg_should_be_built_check
        !last_stage.image.tagged?
      end
    end
  end # Dimg
end # Dapp
