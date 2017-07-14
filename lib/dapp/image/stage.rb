module Dapp
  # Image
  module Image
    # Stage
    class Stage < Docker
      include Argument

      def initialize(name:, project:, built_id: nil, from: nil)
        @container_name = "#{name[/[[^:].]*/]}.#{SecureRandom.hex(4)}"
        @built_id = built_id

        @bash_commands          = []
        @options                = {}
        @change_options         = {}
        @service_change_options = {}

        super(name: name, project: project, from: from)
      end

      def built_id
        @built_id ||= id
      end

      def build!
        run!
        @built_id = commit!
      ensure
        project.shellout("docker rm #{container_name}")
      end

      def built?
        !@built_id.nil?
      end

      def export!(name)
        tag!(name).tap do |image|
          image.push!
          image.untag!
        end
      end

      def tag!(name)
        clone!(name).tap do |image|
          self.class.tag!(id: image.built_id, tag: image.name)
        end
      end

      def import!(name)
        clone!(name).tap do |image|
          image.pull!
          @built_id = image.built_id
          save_in_cache!
          image.untag!
        end
      end

      def save_in_cache!
        project.log_warning(desc: { code: :another_image_already_tagged }) if !(existed_id = id).nil? && built_id != existed_id
        self.class.tag!(id: built_id, tag: name)
      end

      def labels
        raise Error::Build, code: :image_not_exist, data: { name: name } if built_id.nil?
        self.class.image_config_option(image_id: built_id, option: 'labels')
      end

      protected

      attr_reader :container_name

      def run!
        raise Error::Build, code: :built_id_not_defined if from.built_id.nil?
        project.shellout!("docker run #{prepared_options} #{from.built_id} -ec '#{prepared_bash_command}'", log_verbose: true)
      rescue Error::Shellout => error
        project.log_warning(desc: { code: :launched_command, data: { command: prepared_commands.join(' && ') }, context: :container })

        built_id = project.introspect_error? ? commit! : from.built_id
        raise Error::ImageBuildFailed, data: { built_id: built_id,
                                               build_log: error.net_status[:data][:stream],
                                               options: prepared_options }
      end

      def commit!
        project.shellout!("docker commit #{prepared_change} #{container_name}").stdout.strip
      end

      def clone!(name)
        self.class.new(name: name, project: project, built_id: built_id)
      end
    end # Stage
  end # Image
end # Dapp
