module Dapp
  module Dimg
    module Image
      module Argument
        def add_change_volume(value)
          add_change_option(:volume, value)
        end

        def add_change_expose(value)
          add_change_option(:expose, value)
        end

        def add_change_env(**options)
          add_change_option(:env, options)
        end

        def add_change_label(**options)
          add_change_option(:label, options)
        end

        def add_change_cmd(value)
          add_change_option(:cmd, value)
        end

        def add_change_entrypoint(value)
          add_change_option(:entrypoint, value)
        end

        def add_change_onbuild(value)
          add_change_option(:onbuild, value)
        end

        def add_change_workdir(value)
          add_change_option(:workdir, value)
        end

        def add_change_user(value)
          add_change_option(:user, value)
        end

        def add_service_change_label(**options)
          add_service_change_option(:label, options)
        end

        def add_env(var, value)
          add_option(:env, "#{var}=#{value}")
        end

        def add_volume(value)
          add_option(:volume, value)
        end

        def add_volumes_from(value)
          add_option(:'volumes-from', value)
        end

        def add_command(*commands)
          @bash_commands.concat(commands.flatten)
        end

        def add_service_command(*commands)
          @service_bash_commands.concat(commands.flatten)
        end

        def prepare_instructions(options)
          options.map do |key, vals|
            case key
            when :cmd, :entrypoint then [vals]
            when :env, :label then vals.map(&method(:options_to_args)).flatten
            else vals
            end.map { |val| %(#{key.to_s.upcase} #{val}) }
          end.flatten
        end

        protected

        attr_reader :bash_commands, :service_bash_commands
        attr_reader :change_options, :service_change_options
        attr_reader :options

        def image_run_options
          options.tap do |opts|
            opts[:image]      = from.built_id
            opts[:name]       = container_name
            opts[:entrypoint] = dapp.bash_bin
            opts[:cmd]        = ['-ec', prepared_bash_command]
            opts[:hostconfig] = {}.tap do |hostconfig|
              hostconfig[:env] = Array(opts.delete(:'env'))
              hostconfig[:mounts] = [].tap do |mounts|
                Array(opts.delete(:volume)).each do |value|
                  source, target = value.split(':')
                  mounts << { source: source, target: target, type: :bind }
                end
              end

              hostconfig[:volumesfrom] = Array(opts.delete(:'volumes-from'))
            end
          end
        end

        def image_introspect_options
          options.map { |key, vals| Array(vals).map { |val| "--#{key}=#{val}" } }.flatten.join(' ')
        end

        def container_commit_options
          {
            name: container_name,
            changes: prepared_change
          }
        end

        def add_option(key, value)
          add_option_default(options, key, value)
        end

        def add_change_option(key, value)
          add_option_default(change_options, key, value)
        end

        def add_service_change_option(key, value)
          add_option_default(service_change_options, key, value)
        end

        def add_option_default(hash, key, value)
          hash[key] = (hash[key].nil? ? [value] : (hash[key] << value)).flatten
        end

        def from_change_options
          return {} if from.nil?
          [:entrypoint, :cmd].each_with_object({}) do |option, options|
            options[option] = self.class.image_config_option(image_id: from.built_id, option: option)
          end
        end

        def options_to_args(options)
          options.map { |key, value| "#{key}=#{value}" }
        end

        def all_bash_commands
          Array(bash_commands) + Array(service_bash_commands)
        end

        def prepared_change
          prepare_instructions(all_change_options).join("\n")
        end

        def all_change_options
          from_change_options.merge(change_options.merge(service_change_options) { |_, v1, v2| [v1, v2].flatten })
        end

        def prepared_bash_command
          dapp.shellout_pack prepared_commands.join(' && ')
        end

        def prepared_commands
          return [dapp.true_bin] if all_bash_commands.empty?
          all_bash_commands
        end
      end
    end # Image
  end # Dimg
end # Dapp
