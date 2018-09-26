module Dapp::Kube::CLI::Command
  class Kube < ::Dapp::CLI
    class Deploy < Base
      banner <<BANNER.freeze
Usage:

  dapp deployment deploy [options] REPO

Options:
BANNER
      extend ::Dapp::CLI::Options::Tag
      extend ::Dapp::CLI::Options::Ssh

      option :namespace,
             long: '--namespace NAME',
             default: nil

      option :context,
             long: '--context NAME',
             default: nil

      option :image_version,
             long: '--image-version TAG',
             description: "Custom tag (alias for --tag)",
             default: [],
             proc: proc { |v| composite_options(:image_versions) << v }

      option :tmp_dir_prefix,
             long: '--tmp-dir-prefix PREFIX',
             description: 'Tmp directory prefix (/tmp by default). Used for build process service directories.'

      option :helm_set_options,
             long: '--set STRING_ARRAY',
             default: [],
             proc: proc { |v| composite_options(:helm_set) << v }

      option :helm_values_options,
             long: '--values FILE_PATH',
             default: [],
             proc: proc { |v| composite_options(:helm_values) << v }

      option :helm_secret_values_options,
             long: '--secret-values FILE_PATH',
             default: [],
             proc: proc { |v| composite_options(:helm_secret_values) << v }

      option :timeout,
             long: '--timeout INTEGER_SECONDS',
             default: nil,
             description: 'Default timeout to wait for resources to become ready, 300 seconds by default',
             proc: proc {|v| Integer(v)}

      option :kubernetes_timeout,
             long: '--kubernetes-timeout TIMEOUT',
             description: 'Kubernetes api-server tcp connection, read and write timeout (in seconds)',
             proc: ->(v) { v.to_i }

      option :registry_username,
             long: '--registry-username USERNAME'

      option :registry_password,
             long: '--registry-password PASSWORD'

      option :without_registry,
             long: "--without-registry",
             default: false,
             boolean: true,
             description: "Do not connect to docker registry to obtain docker image ids of dimgs being deployed"

      def run(argv = ARGV)
        self.class.parse_options(self, argv)

        options = cli_options
        options[:tag] = [*options.delete(:tag), *options.delete(:image_version)]
        options[:repo] = self.class.required_argument(self, 'repo')
        run_dapp_command(run_method, options: options)
      end
    end
  end
end
