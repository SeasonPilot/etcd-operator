package file

import "net/url"

func ParsePath(rawURL string) (string, string, string, error) {
	URL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", err
	}
	return URL.Scheme, URL.Host, URL.Path, nil
}
