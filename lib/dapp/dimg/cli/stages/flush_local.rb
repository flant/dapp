module Dapp
  module Dimg
    module CLI
      class Stages
        class FlushLocal < Base
          banner <<BANNER.freeze
Usage:

  dapp dimg stages flush local [options] [DIMG ...]

    DIMG                        Dapp image to process [default: *].

Options:
BANNER
          def run(argv = ARGV)
            self.class.parse_options(self, argv)
            ::Dapp::Dapp.new(cli_options: config, dimgs_patterns: cli_arguments).stages_flush_local
          end
        end
      end
    end
  end
end
