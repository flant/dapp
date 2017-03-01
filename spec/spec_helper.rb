require 'bundler/setup'
require 'test_construct/rspec_integration'

if ENV['CODECLIMATE_REPO_TOKEN']
  require 'codeclimate-test-reporter'
  CodeClimate::TestReporter.start
end

Bundler.require :default, :test, :development

require 'active_support'
require 'recursive_open_struct'
require 'tmpdir'

require 'spec_helper/common'
require 'spec_helper/dimg'
require 'spec_helper/git'
require 'spec_helper/config'

RSpec.configure do |config|
  config.before :all do
    Dapp::Dapp::Logging::I18n.initialize

    # Force /tmp as base dir for all mktmpdir calls.
    # Needed to enable macos rspec tests.
    # By default on macos tmp-dirs are stored in /var/folder.
    # That causes docker mounts problems with default macos docker file sharing settings.
    ::Dir.define_singleton_method(:tmpdir) {'/tmp'}
  end
  config.mock_with :rspec do |mocks|
    mocks.allow_message_expectations_on_nil = true
  end
end
