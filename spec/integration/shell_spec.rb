require_relative '../spec_helper'

describe Dapp::Dimg::Builder::Shell do
  include SpecHelper::Common
  include SpecHelper::Dimg

  def config
    @config ||= default_config.merge(_builder: :shell)
  end

  def expect_files
    image_name = stages[:g_a_latest_patch].send(:image_name)
    config[:_shell].keys.each do |stage|
      expect { shellout!("docker run --rm #{image_name} bash -lec 'cat /#{stage}'") }.to_not raise_error
    end
  end

  %w(ubuntu:14.04 centos:7).each do |image|
    it "build #{image}" do
      config[:_docker][:_from] = image
      config[:_shell].keys.each { |stage| config[:_shell][stage] << "date +%s > /#{stage}" }
      dimg_build!

      expect_files
    end
  end
end
