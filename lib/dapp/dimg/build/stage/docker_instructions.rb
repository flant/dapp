module Dapp
  module Dimg
    module Build
      module Stage
        # DockerInstructions
        class DockerInstructions < Base
          def initialize(dimg)
            @prev_stage = GALatestPatch.new(dimg, self)
            @dimg = dimg
          end

          def dependencies
            [change_options]
          end

          def prepare_image
            super
            change_options.each do |k, v|
              image.public_send("add_change_#{k}", v)
            end
          end
        end # DockerInstructions
      end # Stage
    end # Build
  end # Dimg
end # Dapp
