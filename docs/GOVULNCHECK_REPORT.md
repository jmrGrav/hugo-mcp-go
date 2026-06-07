# Govulncheck Report

## Failing run

Command:

```bash
PATH="$HOME/go/bin:$HOME/.local/bin:$PATH" govulncheck ./...
```

Toolchain at failing time:

- `go version go1.25.0 linux/amd64`
- `go env GOVERSION=go1.25.0`
- `go env GOTOOLCHAIN=auto`
- `go env GOROOT=/home/jm/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64`

Raw output:

```text
=== Symbol Results ===

Vulnerability #1: GO-2026-5039
    Arbitrary inputs are included in errors without any escaping in
    net/textproto
  More info: https://pkg.go.dev/vuln/GO-2026-5039
  Standard library
    Found in: net/textproto@go1.25
    Fixed in: net/textproto@go1.25.11
    Example traces found:
      #1: internal/hugo/mutations/pages.go:225:24: mutations.PageService.Delete calls io.ReadAll, which eventually calls textproto.Reader.ReadMIMEHeader

Vulnerability #2: GO-2026-5037
    Inefficient candidate hostname parsing in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2026-5037
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.11
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify
      #2: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.VerifyHostname
      #3: cmd/hugo-mcp-go/main.go:17:83: hugo.main calls x509.HostnameError.Error

Vulnerability #3: GO-2026-4971
    Panic in Dial and LookupPort when handling NUL byte on Windows in net
  More info: https://pkg.go.dev/vuln/GO-2026-4971
  Standard library
    Found in: net@go1.25
    Fixed in: net@go1.25.10
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls net.Dialer.DialContext
      #2: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which calls net.Listen

Vulnerability #4: GO-2026-4947
    Unexpected work during chain building in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2026-4947
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.9
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify

Vulnerability #5: GO-2026-4946
    Inefficient policy validation in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2026-4946
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.9
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify

Vulnerability #6: GO-2026-4918
    Infinite loop in HTTP/2 transport when given bad SETTINGS_MAX_FRAME_SIZE in
    net/http/internal/http2 in golang.org/x/net
  More info: https://pkg.go.dev/vuln/GO-2026-4918
  Standard library
    Found in: net/http@go1.25
    Fixed in: net/http@go1.25.10
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls http.Client.Do

Vulnerability #7: GO-2026-4870
    Unauthenticated TLS 1.3 KeyUpdate record can cause persistent connection
    retention and DoS in crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2026-4870
  Standard library
    Found in: crypto/tls@go1.25
    Fixed in: crypto/tls@go1.25.9
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Conn.HandshakeContext
      #2: internal/hugo/mutations/pages.go:225:24: mutations.PageService.Delete calls io.ReadAll, which eventually calls tls.Conn.Read
      #3: cmd/hugo-mcp-shim/main.go:18:14: hugo.main calls fmt.Fprintf, which calls tls.Conn.Write
      #4: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Dialer.DialContext

Vulnerability #8: GO-2026-4602
    FileInfo can escape from a Root in os
  More info: https://pkg.go.dev/vuln/GO-2026-4602
  Standard library
    Found in: os@go1.25
    Fixed in: os@go1.25.8
    Example traces found:
      #1: internal/hugo/pages/pages.go:230:28: pages.collectDirectIndex calls os.ReadDir

Vulnerability #9: GO-2026-4601
    Incorrect parsing of IPv6 host literals in net/url
  More info: https://pkg.go.dev/vuln/GO-2026-4601
  Standard library
    Found in: net/url@go1.25
    Fixed in: net/url@go1.25.8
    Example traces found:
      #1: internal/tools/tools.go:85:13: tools.Register calls mcp.AddTool[github.com/jmrGrav/hugo-mcp-go/internal/tools.listPagesInput github.com/jmrGrav/hugo-mcp-go/internal/tools.listPagesOutput], which eventually calls url.Parse
      #2: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls url.ParseRequestURI
      #3: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls url.URL.Parse

Vulnerability #10: GO-2026-4340
    Handshake messages may be processed at the incorrect encryption level in
    crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2026-4340
  Standard library
    Found in: crypto/tls@go1.25
    Fixed in: crypto/tls@go1.25.6
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Conn.HandshakeContext
      #2: internal/hugo/mutations/pages.go:225:24: mutations.PageService.Delete calls io.ReadAll, which eventually calls tls.Conn.Read
      #3: cmd/hugo-mcp-shim/main.go:18:14: hugo.main calls fmt.Fprintf, which calls tls.Conn.Write
      #4: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Dialer.DialContext

Vulnerability #11: GO-2026-4337
    Unexpected session resumption in crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2026-4337
  Standard library
    Found in: crypto/tls@go1.25
    Fixed in: crypto/tls@go1.25.7
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Conn.HandshakeContext
      #2: internal/hugo/mutations/pages.go:225:24: mutations.PageService.Delete calls io.ReadAll, which eventually calls tls.Conn.Read
      #3: cmd/hugo-mcp-shim/main.go:18:14: hugo.main calls fmt.Fprintf, which calls tls.Conn.Write
      #4: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Dialer.DialContext

Vulnerability #12: GO-2025-4175
    Improper application of excluded DNS name constraints when verifying
    wildcard names in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4175
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.5
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify

Vulnerability #13: GO-2025-4155
    Excessive resource consumption when printing error string for host
    certificate validation in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4155
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.5
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify
      #2: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.VerifyHostname

Vulnerability #14: GO-2025-4013
    Panic when validating certificates with DSA public keys in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4013
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.2
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify

Vulnerability #15: GO-2025-4012
    Lack of limit when parsing cookies can cause memory exhaustion in net/http
  More info: https://pkg.go.dev/vuln/GO-2025-4012
  Standard library
    Found in: net/http@go1.25
    Fixed in: net/http@go1.25.2
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls http.Client.Do

Vulnerability #16: GO-2025-4011
    Parsing DER payload can cause memory exhaustion in encoding/asn1
  More info: https://pkg.go.dev/vuln/GO-2025-4011
  Standard library
    Found in: encoding/asn1@go1.25
    Fixed in: encoding/asn1@go1.25.2
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls asn1.Unmarshal

Vulnerability #17: GO-2025-4010
    Insufficient validation of bracketed IPv6 hostnames in net/url
  More info: https://pkg.go.dev/vuln/GO-2025-4010
  Standard library
    Found in: net/url@go1.25
    Fixed in: net/url@go1.25.2
    Example traces found:
      #1: internal/tools/tools.go:85:13: tools.Register calls mcp.AddTool[github.com/jmrGrav/hugo-mcp-go/internal/tools.listPagesInput github.com/jmrGrav/hugo-mcp-go/internal/tools.listPagesOutput], which eventually calls url.Parse
      #2: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls url.ParseRequestURI
      #3: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls url.URL.Parse

Vulnerability #18: GO-2025-4009
    Quadratic complexity when parsing some invalid inputs in encoding/pem
  More info: https://pkg.go.dev/vuln/GO-2025-4009
  Standard library
    Found in: encoding/pem@go1.25
    Fixed in: encoding/pem@go1.25.2
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls pem.Decode

Vulnerability #19: GO-2025-4008
    ALPN negotiation error contains attacker controlled information in
    crypto/tls
  More info: https://pkg.go.dev/vuln/GO-2025-4008
  Standard library
    Found in: crypto/tls@go1.25
    Fixed in: crypto/tls@go1.25.2
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Conn.HandshakeContext
      #2: internal/hugo/mutations/pages.go:225:24: mutations.PageService.Delete calls io.ReadAll, which eventually calls tls.Conn.Read
      #3: cmd/hugo-mcp-shim/main.go:18:14: hugo.main calls fmt.Fprintf, which calls tls.Conn.Write
      #4: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls tls.Dialer.DialContext

Vulnerability #20: GO-2025-4007
    Quadratic complexity when checking name constraints in crypto/x509
  More info: https://pkg.go.dev/vuln/GO-2025-4007
  Standard library
    Found in: crypto/x509@go1.25
    Fixed in: crypto/x509@go1.25.3
    Example traces found:
      #1: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.CertPool.AppendCertsFromPEM
      #2: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.Certificate.Verify
      #3: internal/shim/server.go:57:32: shim.Server.ListenAndServe calls http.Server.ListenAndServe, which eventually calls x509.ParseCertificate

Your code is affected by 20 vulnerabilities from the Go standard library.
This scan also found 6 vulnerabilities in packages you import and 12
vulnerabilities in modules you require, but your code doesn't appear to call
these vulnerabilities.
Use '-show verbose' for more details.

```

## Patched verification

Command:

```bash
GOTOOLCHAIN=go1.25.11 PATH="$HOME/go/bin:$HOME/.local/bin:$PATH" govulncheck ./...
```

Result:

```text
No vulnerabilities found.
Your code is affected by 0 vulnerabilities.
This scan also found 0 vulnerabilities in packages you import and 1 vulnerability in modules you require, but your code does not appear to call these vulnerabilities.
```
