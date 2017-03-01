module Dapp
  class Dapp
    module Shellout
      # Streaming
      module Streaming
        # Stream
        class Stream
          def buffer
            @buffer ||= []
          end

          def <<(string)
            buffer << string
          end

          def show
            buffer.join.strip
          end
        end

        # Proxy
        module Proxy
          # Base
          class Base
            def initialize(*streams, with_time: false)
              @streams = streams
              @with_time = with_time
            end

            def <<(str)
              @streams.each { |s| s << format_string(str) }
            end

            def format_string(str)
              str.lines.map { |l| "#{Dapp.log_time if @with_time}#{l.strip}\n" }.join
            end
          end

          # Error
          class Error < Base
            def format_string(str)
              "#{Dapp.paint_string(super.strip, :warning)}\n"
            end
          end
        end
      end
    end
  end # Helper
end # Dapp
