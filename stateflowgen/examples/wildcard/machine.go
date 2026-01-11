package wildcard

//go:generate go run github.com/donutnomad/gogen gen ./...

// 通配符测试：* 符号
// Ready(*) 会展开为 Ready(Running) 和 Ready(Stopped)
// 通配符展开时不包含自我流转
// @StateFlow(name="Machine")
// @Flow: Init            => [ Ready(Running) ]
// @Flow: Ready(Running)  => [ (Stopped) ]
// @Flow: Ready(Stopped)  => [ (Running) ]
// @Flow: Ready(*)        => [ Terminated! via Terminating ]
const machineStateFlow = ""
