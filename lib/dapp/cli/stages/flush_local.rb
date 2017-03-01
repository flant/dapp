module Dapp
  class CLI
    class Stages
      # stages flush local subcommand
      class FlushLocal < Base
        banner <<BANNER.freeze
Version: #{Dapp::VERSION}

Usage:
  dapp stages flush local [options] [DIMG ...]

    DIMG                        Dapp image to process [default: *].

Options:
BANNER
        def run(argv = ARGV)
          self.class.parse_options(self, argv)
          Dapp.new(cli_options: config, dimgs_patterns: cli_arguments).stages_flush_local
        end
      end
    end
  end
end
