// +build !ENABLE_TRACE

package util

/*
	TraceK functions written below are generated by the following Ruby script:

		def foo(i)
		  args = i.times.map { |k| "v#{k}" }.join(", ")
		  if i == 0
			puts "func Trace0(format string) {"
		  else
			puts "func Trace#{i}(format string, #{args} interface{}) {"
		  end
		  puts "}"
		end
		(0...10).each { |i| foo(i); puts "\n" }
*/

func Trace0(format string) {
}

func Trace1(format string, v0 interface{}) {
}

func Trace2(format string, v0, v1 interface{}) {
}

func Trace3(format string, v0, v1, v2 interface{}) {
}

func Trace4(format string, v0, v1, v2, v3 interface{}) {
}

func Trace5(format string, v0, v1, v2, v3, v4 interface{}) {
}

func Trace6(format string, v0, v1, v2, v3, v4, v5 interface{}) {
}

func Trace7(format string, v0, v1, v2, v3, v4, v5, v6 interface{}) {
}

func Trace8(format string, v0, v1, v2, v3, v4, v5, v6, v7 interface{}) {
}

func Trace9(format string, v0, v1, v2, v3, v4, v5, v6, v7, v8 interface{}) {
}
