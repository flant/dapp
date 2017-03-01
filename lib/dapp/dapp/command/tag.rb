module Dapp
  # Dapp
  class Dapp
    # Command
    module Command
      # Tag
      module Tag
        def tag(tag)
          one_dimg!
          raise Error::Dapp, code: :tag_command_incorrect_tag, data: { name: tag } unless Image::Docker.image_name?(tag)
          Dimg::Dimg.new(config: build_configs.first, dapp: self, ignore_git_fetch: true, should_be_built: true).tap do |app|
            app.tag!(tag)
          end
        end
      end
    end
  end # Dapp
end # Dapp
