package platform

const CoreSupportedPlatformsURL = "https://core.productive-k3s.io/en/product/supported-platforms/"

func SupportsCoreHost(goos, goarch string) bool {
	return goos == "linux" && goarch == "amd64"
}
