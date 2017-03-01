module Dapp
  class CLI
    class Stages
      # stages pull subcommand
      class Pull < Base
        banner <<BANNER.freeze
Version: #{Dapp::VERSION}

Usage:
  dapp stages pull [options] [DIMG ...] REPO

    DIMG                        Dapp image to process [default: *].

Options:
BANNER
        option :pull_all_stages,
               long: '--all',
               boolean: true

        def run(argv = ARGV)
          self.class.parse_options(self, argv)
          repo = self.class.required_argument(self)
          Dapp.new(cli_options: config, dimgs_patterns: cli_arguments).stages_pull(repo)
        end
      end
    end
  end
end
