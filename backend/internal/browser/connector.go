package browser

// BuildLaunchArgs 构建启动参数
func BuildLaunchArgs(args []string, profile *Profile, defaultStartURLs []string) []string {
	args = append(args, defaultStartURLs...)
	return args
}
