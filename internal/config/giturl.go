package config

import (
	"encoding/base64"
	"strings"
)

const (
	githubSCPPrefix = "git@github.com:"       // git@github.com:owner/repo.git
	githubSSHPrefix = "ssh://git@github.com/" // ssh://git@github.com/owner/repo.git
	githubHTTPSBase = "https://github.com/"
)

// NormalizeGitHubCloneURL rewrites a GitHub SSH config-repo URL to its HTTPS
// equivalent. A freshly created sprite has no SSH key registered with GitHub and
// no github.com entry in known_hosts, so an SSH clone (git@github.com:owner/repo)
// fails with "Host key verification failed" before the ssh_setup phase (which
// would provision both) can run. The HTTPS form needs neither: public repos
// clone anonymously, and private repos clone with the forwarded GH_TOKEN (see
// GitHubHTTPSAuthArgs).
//
// Only github.com SSH URLs are rewritten. HTTPS URLs, other hosts, and local
// paths are returned unchanged with rewritten=false.
func NormalizeGitHubCloneURL(url string) (normalized string, rewritten bool) {
	switch {
	case strings.HasPrefix(url, githubSCPPrefix):
		return githubHTTPSBase + strings.TrimPrefix(url, githubSCPPrefix), true
	case strings.HasPrefix(url, githubSSHPrefix):
		return githubHTTPSBase + strings.TrimPrefix(url, githubSSHPrefix), true
	case strings.HasPrefix(url, "ssh://git@github.com:"):
		// ssh://git@github.com:22/owner/repo.git: an explicit port follows the
		// host, so strip everything up to the first "/" after the host before
		// rewriting to HTTPS.
		rest := strings.TrimPrefix(url, "ssh://git@github.com:")
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			return githubHTTPSBase + rest[i+1:], true
		}
		return url, false
	default:
		return url, false
	}
}

// GitHubHTTPSAuthArgs returns transient `git -c` arguments that authenticate
// HTTPS requests to github.com with token via an Authorization header, so a
// private config repo can be cloned over HTTPS using the forwarded GH_TOKEN.
// Passing the credential with -c (rather than embedding it in the remote URL)
// keeps the token out of the cloned repo's .git/config on disk.
//
// Returns nil when token is empty or url is not an https://github.com/ URL, so
// the public-repo path adds no arguments and forwards no header.
func GitHubHTTPSAuthArgs(token, url string) []string {
	if token == "" || !strings.HasPrefix(url, githubHTTPSBase) {
		return nil
	}
	cred := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + token))
	return []string{"-c", "http." + githubHTTPSBase + ".extraheader=AUTHORIZATION: basic " + cred}
}
