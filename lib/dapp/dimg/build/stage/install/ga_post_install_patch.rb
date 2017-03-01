module Dapp
  module Dimg
    module Build
      module Stage
        module InstallGroup
          # GAPostInstallPatch
          class GAPostInstallPatch < GABase
            include Mod::Group

            def initialize(dimg, next_stage)
              @prev_stage = GAPostInstallPatchDependencies.new(dimg, self)
              super
            end

            def next_g_a_stage
              super.next_stage.next_stage # GAPreSetupPatch
            end
          end # GAPostInstallPatch
        end
      end # Stage
    end # Build
  end # Dimg
end # Dapp
