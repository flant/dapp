module Dapp
  module Kube
    module Dapp
      module Command
        module Common
          def kube_check_helm!
            raise Error::Command, code: :helm_not_found if shellout('which helm').exitstatus == 1
          end

          def kube_release_name
            "#{name}-#{kube_namespace}"
          end

          def kube_namespace
            kubernetes.namespace
          end

          def kube_helm_decode_json(json)
            decode_value = proc do |value|
              case value
              when Array then value.map { |v| decode_value.call(v) }
              when Hash then kube_helm_decode_json(value)
              else
                secret.nil? ? '' : secret.extract(value)
              end
            end
            json.each { |k, v| json[k] = decode_value.call(v) }
          end

          def secret_key_should_exist!
            raise(Error::Command,
              code: :secret_key_not_found,
              data: {not_found_in: secret_key_not_found_in.join(', ')}
            ) if secret.nil?
          end

          def secret
            @secret ||= begin
              unless secret_key = ENV['DAPP_SECRET_KEY']
                secret_key_not_found_in << '`DAPP_SECRET_KEY`'

                if dappfile_exists?
                  file_path = path('.dapp_secret_key')
                  if file_path.file?
                    secret_key = path('.dapp_secret_key').read.chomp
                  else
                    secret_key_not_found_in << "`#{file_path}`"
                  end
                end
              end

              Secret.new(secret_key) if secret_key
            end
          end

          def secret_key_not_found_in
            @secret_key_not_found_in ||= []
          end

          def kubernetes
            @kubernetes ||= begin
              namespace = options[:namespace].nil? ? nil : options[:namespace].tr('_', '-')
              Client.new(namespace: namespace)
            end
          end
        end
      end
    end
  end
end
