require_relative '../spec_helper'

describe Dapp::Dimg do
  include SpecHelper::Common
  include SpecHelper::Dimg
  include SpecHelper::Git

  before :all do
    @wd = Dir.pwd
    init
  end

  before :each do
    # git_init works only in test case context,
    # because of using dapp and dimg objects for system-shellout.
    # But git should only be initialized once.
    # Earlier git_init was in before :all, but that is not possible now.
    self.class.instance_variable_get(:@git_initialized) || begin
      git_init
      self.class.instance_variable_set(:@git_initialized, true)
    end

    dimg_build!
  end

  after :all do
    Dir.chdir @wd
  end

  def init
    FileUtils.rm_rf project_path
    FileUtils.mkpath project_path
    Dir.chdir project_path
  end

  def project_path
    Pathname('/tmp/dapp/test')
  end

  def config
    @config ||= default_config.merge(
      _builder: :shell,
      _home_path: project_path,
      _docker: default_config[:_docker].merge(_from: :'ubuntu:16.04'),
      _git_artifact: default_config[:_git_artifact].merge(_local: { _artifact_options: { to: '/to', exclude_paths: [] } })
    )
  end

  def stages_names
    @stages ||= stages.keys.reverse
  end

  def stage_index(stage_name)
    stages_names.index(stage_name)
  end

  def prev_stage(s)
    stages[s].prev_stage.send(:name)
  end

  def stages_signatures
    stages.values.map { |s| [:"#{s.send(:name)}", s.send(:signature)] }.to_h
  end

  def check_image_command(stage_name, command)
    expect(stages[stage_name].send(:image).send(:bash_commands).join =~ Regexp.new(command)).to be
  end

  def expect_from_image
  end

  def expect_g_a_archive_image
    check_image_command(:g_a_archive, 'tar -x')
  end

  def expect_before_install_image
    check_image_command(:before_install, config[:_shell][:_before_install_command].last)
    check_image_command(:g_a_archive, 'tar -x')
  end

  def expect_before_setup_image
    check_image_command(:before_setup, config[:_shell][:_before_setup_command].last)
    check_image_command(:g_a_post_install_patch, 'apply')
  end

  [:install, :setup].each do |stage_name|
    define_method "expect_#{stage_name}_image" do
      check_image_command(stage_name, config[:_shell][:"_#{stage_name}_command"].last)
      check_image_command(prev_stage(stage_name), 'apply')
    end
  end

  [:g_a_post_setup_patch, :g_a_latest_patch].each do |stage_name|
    define_method "expect_#{stage_name}_image" do
      check_image_command(stage_name, 'apply')
    end
  end

  def change_from
    config[:_docker][:_from] = :'ubuntu:14.04'
  end

  [:before_install, :install, :before_setup, :setup].each do |stage_name|
    define_method :"change_#{stage_name}" do
      config[:_shell][:"_#{stage_name}_command"] << generate_command
    end
  end

  def change_g_a_archive
    git_change_and_commit(msg: Dapp::Dimg::Build::Stage::GAArchiveDependencies::RESET_COMMIT_MESSAGES.sample)
  end

  def change_g_a_post_setup_patch
    git_change_and_commit('large_file', random_string(Dapp::Dimg::Build::Stage::SetupGroup::GAPostSetupPatchDependencies::MAX_PATCH_SIZE))
  end

  def change_g_a_latest_patch
    git_change_and_commit
  end

  def from_modified_signatures
    stages_names
  end

  def install_modified_signatures
    stages_names[stage_index(:g_a_pre_install_patch_dependencies)..-1]
  end

  def before_setup_modified_signatures
    stages_names[stage_index(:g_a_post_install_patch_dependencies)..-1]
  end

  def setup_modified_signatures
    stages_names[stage_index(:g_a_pre_setup_patch_dependencies)..-1]
  end

  [:before_install, :g_a_archive, :g_a_post_setup_patch, :g_a_latest_patch].each do |stage_name|
    define_method "#{stage_name}_modified_signatures" do
      stages_names[stage_index(stage_name)..-1]
    end
  end

  def from_saved_signatures
    []
  end

  def before_install_saved_signatures
    [stages_names.first]
  end

  def g_a_archive_saved_signatures
    stages_names[0..stage_index(:before_install)]
  end

  def install_saved_signatures
    stages_names[0..stage_index(:g_a_archive)]
  end

  def before_setup_saved_signatures
    stages_names[0..stage_index(:install)]
  end

  def setup_saved_signatures
    stages_names[0..stage_index(:before_setup)]
  end

  def g_a_post_setup_patch_saved_signatures
    stages_names[0..stage_index(:setup)]
  end

  def g_a_latest_patch_saved_signatures
    stages_names[0..stage_index(:g_a_post_setup_patch)]
  end

  def build_and_check(stage_name)
    check_signatures_and_build(stage_name)
    send("expect_#{stage_name}_image")
  end

  def check_signatures_and_build(stage_name)
    saved_signatures = stages_signatures
    send(:"change_#{stage_name}")
    dimg_renew
    expect_stages_signatures(stage_name, saved_signatures, stages_signatures)
    dimg_build!
  end

  def expect_stages_signatures(stage_name, saved_keys, new_keys)
    send("#{stage_name}_saved_signatures").each { |s| expect(saved_keys).to include s => new_keys[s] }
    send("#{stage_name}_modified_signatures").each { |s| expect(saved_keys).to_not include s => new_keys[s] }
  end

  def g_a_latest_patch
    build_and_check(:g_a_latest_patch)
  end

  def g_a_post_setup_patch
    build_and_check(:g_a_post_setup_patch)
    g_a_latest_patch
  end

  def setup
    build_and_check(:setup)
    g_a_latest_patch
    g_a_post_setup_patch
  end

  def before_setup
    build_and_check(:before_setup)
    g_a_latest_patch
    g_a_post_setup_patch
    setup
  end

  def install
    build_and_check(:install)
    g_a_latest_patch
    g_a_post_setup_patch
    setup
    before_setup
  end

  def g_a_archive
    build_and_check(:g_a_archive)
    g_a_latest_patch
    g_a_post_setup_patch
    setup
    before_setup
    install
  end

  def before_install
    build_and_check(:before_install)
    g_a_latest_patch
    g_a_post_setup_patch
    setup
    before_setup
    install
    g_a_archive
  end

  def from
    build_and_check(:from)
    g_a_latest_patch
    g_a_post_setup_patch
    setup
    before_setup
    install
    g_a_archive
    before_install
  end

  [:g_a_latest_patch, :g_a_post_setup_patch, :setup, :before_setup, :install, :g_a_archive, :before_install, :from].each do |stage|
    it "test #{stage}" do
      progress_thr = nil
      progress_thr = Thread.new {
        STDOUT.sync = true
        STDERR.sync = true
        loop { sleep(60); puts '.' }
      } if ENV['TRAVIS']

      begin
        send(stage)
      ensure
        progress_thr.kill if progress_thr
      end
    end
  end
end
