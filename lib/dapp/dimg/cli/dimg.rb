module Dapp
  module Dimg
    module CLI
      class Dimg < ::Dapp::CLI
        SUBCOMMANDS = ['build', 'push', 'spush', 'list', 'run', 'stages', 'cleanup', 'bp', 'mrproper', 'stage image', 'tag'].freeze

        banner <<BANNER.freeze
Usage: dapp dimg [options] sub-command [sub-command options]

Available subcommands: (for details, dapp dimg SUB-COMMAND --help)

dapp dimg build [options] [DIMG ...]
dapp dimg bp [options] [DIMG ...] REPO
dapp dimg push [options] [DIMG ...] REPO
dapp dimg spush [options] [DIMG] REPO
dapp dimg tag [options] [DIMG] TAG
dapp dimg list [options] [DIMG ...]
dapp dimg run [options] [DIMG] [DOCKER ARGS]
dapp dimg cleanup [options] [DIMG ...]
dapp dimg mrproper [options]
dapp dimg stage image [options] [DIMG]
dapp dimg stages

Options:
BANNER

      end
    end
  end
end

::Dapp::CLI.send(:include, ::Dapp::Dimg::CLI)