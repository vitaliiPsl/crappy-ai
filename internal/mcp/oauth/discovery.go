package oauth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

type Discovery struct {
	Resource         string
	AuthorizationURL string
	TokenURL         string
	RegistrationURL  string
	Scopes           []string
}

type resourceCandidate struct {
	url      string
	resource string
}

func Discover(ctx context.Context, client *http.Client, serverURL string, resp *http.Response) (Discovery, error) {
	metadataURL := readChallenge(resp)

	prm, err := fetchResourceMetadata(ctx, client, serverURL, metadataURL)
	if err != nil {
		return Discovery{}, err
	}

	asm, err := fetchAuthServerMetadata(ctx, client, prm.AuthorizationServers)
	if err != nil {
		return Discovery{}, err
	}

	return Discovery{
		Resource:         prm.Resource,
		AuthorizationURL: asm.AuthorizationEndpoint,
		TokenURL:         asm.TokenEndpoint,
		RegistrationURL:  asm.RegistrationEndpoint,
		Scopes:           prm.ScopesSupported,
	}, nil
}

func fetchResourceMetadata(
	ctx context.Context,
	client *http.Client,
	serverURL string,
	metadataURL string,
) (*oauthex.ProtectedResourceMetadata, error) {
	for _, candidate := range resourceMetadataURLs(metadataURL, serverURL) {
		prm, err := oauthex.GetProtectedResourceMetadata(ctx, candidate.url, candidate.resource, client)
		if err != nil || prm == nil {
			continue
		}

		if len(prm.AuthorizationServers) == 0 {
			return nil, errors.New("oauth: protected resource metadata has no authorization servers")
		}

		return prm, nil
	}

	return nil, fmt.Errorf("oauth: no protected resource metadata for %q", serverURL)
}

func fetchAuthServerMetadata(ctx context.Context, client *http.Client, issuers []string) (*oauthex.AuthServerMeta, error) {
	var lastErr error

	for _, issuer := range issuers {
		for _, metadataURL := range authServerMetadataURLs(issuer) {
			metadata, err := oauthex.GetAuthServerMeta(ctx, metadataURL, issuer, client)
			if err != nil {
				lastErr = err

				continue
			}

			if metadata != nil {
				return metadata, nil
			}
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, fmt.Errorf("oauth: no authorization server metadata for issuers %v", issuers)
}

func readChallenge(resp *http.Response) string {
	defer closeResponse(resp)

	if resp == nil {
		return ""
	}

	challenges, err := oauthex.ParseWWWAuthenticate(resp.Header.Values("WWW-Authenticate"))
	if err != nil {
		return ""
	}

	for _, challenge := range challenges {
		if challenge.Scheme != "bearer" {
			continue
		}

		if url := challenge.Params["resource_metadata"]; url != "" {
			return url
		}
	}

	return ""
}

func resourceMetadataURLs(metadataURL, resourceURL string) []resourceCandidate {
	var candidates []resourceCandidate

	if metadataURL != "" {
		candidates = append(candidates, resourceCandidate{url: metadataURL, resource: resourceURL})
	}

	rurl, err := url.Parse(resourceURL)
	if err != nil {
		return candidates
	}

	atPath := *rurl
	atPath.Path = "/.well-known/oauth-protected-resource/" + strings.TrimLeft(rurl.Path, "/")
	candidates = append(candidates, resourceCandidate{url: atPath.String(), resource: resourceURL})

	atRoot := *rurl
	atRoot.Path = "/.well-known/oauth-protected-resource"
	rurl.Path = ""
	candidates = append(candidates, resourceCandidate{url: atRoot.String(), resource: rurl.String()})

	return candidates
}

func authServerMetadataURLs(issuer string) []string {
	u, err := url.Parse(issuer)
	if err != nil {
		return nil
	}

	path := strings.TrimSuffix(u.Path, "/")

	rfc8414 := *u
	rfc8414.Path = "/.well-known/oauth-authorization-server" + path

	oidc := *u
	oidc.Path = path + "/.well-known/openid-configuration"

	return []string{rfc8414.String(), oidc.String()}
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)
}
