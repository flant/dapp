module Dapp::Kube::CLI::Command
  class Kube < ::Dapp::CLI
    class ValueGet < Base
      banner <<BANNER.freeze
Usage:

  dapp kube value get [options] VALUE_KEY REPO

Options:
BANNER
      extend ::Dapp::CLI::Options::Tag

      option :namespace,
        long: '--namespace NAME',
        default: nil

      option :context,
        long: '--context NAME',
        default: nil

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
        repo = self.class.required_argument(self, 'repo')
        run_dapp_command(nil, options: cli_options(repo: repo)) do |dapp|
          value_key = self.class.required_argument(self, "VALUE_KEY")
          dapp.public_send(run_method, value_key)
        end
      end
    end

    def log_running_time
      false
    end
  end
end
