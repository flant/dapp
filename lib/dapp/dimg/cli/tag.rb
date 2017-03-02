module Dapp
  module Dimg
    module CLI
      # CLI tag subcommand
      class Tag < Base
        banner <<BANNER.freeze
Version: #{::Dapp::VERSION}

Usage:
  dapp dimg tag [options] [DIMG] TAG

    DIMG                        Dapp image to process [default: *].
    REPO                        Pushed image name.

Options:
BANNER

        def run(argv = ARGV)
          self.class.parse_options(self, argv)
          tag = self.class.required_argument(self)
          Dapp.new(cli_options: config, dimgs_patterns: cli_arguments).public_send(class_to_lowercase, tag)
        end
      end
    end
  end
end
