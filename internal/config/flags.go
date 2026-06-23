package config

func ParseInstallFlags(args []string) (FlagSet, []string) {
	var flags FlagSet
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--platforms":
			if i+1 < len(args) {
				flags.Platforms = args[i+1]
				i++
			}
		case "--docs-mode":
			if i+1 < len(args) {
				flags.DocsMode = args[i+1]
				i++
			}
		case "--path":
			if i+1 < len(args) {
				flags.Path = args[i+1]
				i++
			}
		case "--yes":
			flags.Yes = true
		default:
			remaining = append(remaining, args[i])
		}
	}

	return flags, remaining
}

func HasInstallFlags(f FlagSet) bool {
	return f.Platforms != "" || f.DocsMode != "" || f.Path != "" || f.Yes
}
