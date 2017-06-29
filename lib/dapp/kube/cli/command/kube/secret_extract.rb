module Dapp::Kube::CLI::Command
  class Kube < ::Dapp::CLI
    class SecretExtract < Base
      banner <<BANNER.freeze
Usage:

  dapp kube secret extract [FILE_PATH] [options]

Options:
BANNER

      option :output_file_path,
             short: '-o OUTPUT_FILE_PATH',
             description: 'Output file',
             required: false

      option :values,
             long: '--values',
             description: 'Decode secret values file',
             default: false

      def run(argv = ARGV)
        self.class.parse_options(self, argv)
        file_path = begin
          if cli_options[:values] || !cli_arguments.empty?
            self.class.required_argument(self, 'FILE_PATH')
          end
        end
        ::Dapp::Dapp.new(options: cli_options).public_send(run_method, file_path)
      end
    end
  end
end
