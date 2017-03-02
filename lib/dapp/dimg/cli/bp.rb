module Dapp
  module Dimg
    module CLI
      # CLI bp subcommand
      class Bp < Push
        banner <<BANNER.freeze
Version: #{::Dapp::VERSION}

Usage:
  dapp dimg bp [options] [DIMG ...] REPO

    DIMG                        Dapp image to process [default: *].
    REPO                        Pushed image name.

Options:
BANNER
        option :tmp_dir_prefix,
               long: '--tmp-dir-prefix PREFIX',
               description: 'Tmp directory prefix (/tmp by default). Used for build process service directories.'

        option :lock_timeout,
               long: '--lock-timeout TIMEOUT',
               description: 'Redefine resource locking timeout (in seconds)',
               proc: ->(v) { v.to_i }

        option :git_artifact_branch,
               long: '--git-artifact-branch BRANCH',
               description: 'Default branch to archive artifacts from'

        option :ssh_key,
               long: '--ssh-key SSH_KEY',
               description: ['Enable only specified ssh keys ',
                             '(use system ssh-agent by default)'].join,
               default: nil,
               proc: ->(v) { composite_options(:ssh_key) << v }
      end
    end
  end
end
