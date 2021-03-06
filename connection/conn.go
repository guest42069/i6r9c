package connection

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"

	"golang.org/x/net/proxy"
)

// Login handles sending the initial login sequence to the remote server, including requesting SASL cert auth if required.
func Login(send chan<- string, nick string, sasl bool) {
	if sasl {
		send <- "CAP REQ :sasl"
		send <- "AUTHENTICATE EXTERNAL"
		send <- "AUTHENTICATE +"
		send <- "CAP END"
	}
	send <- fmt.Sprintf("NICK %s", nick)
	send <- fmt.Sprintf("USER %s * * :%s", nick, nick)
}

// Connect will connect to serverAddr, over the proxy at proxyAddr. clientAuthCert will be used as the client certificate auth. tlsVerify will determine if invalid certificates are accepted.
func Connect(proxyAddr, serverAddr *string, clientAuthCert *tls.Certificate, tlsVerify bool) (net.Conn, error) {
	proxyUrl, err := url.Parse(*proxyAddr)
	if err != nil {
		return nil, err
	}
	serverUrl, err := url.Parse(*serverAddr)
	if err != nil {
		return nil, err
	}
	// sorry, if you're not using Tor this is probably inconvenient.
	// however this client is designed for use with Tor.
	// this ensures that circuits are isolated, I.E. two distinct IRC connection won't use the same Tor circuit (linking them together).
	var blob [16]byte
	rand.Read(blob[:])
	var user = fmt.Sprintf("%x", blob[:])
	rand.Read(blob[:])
	var passwd = fmt.Sprintf("%x", blob[:])
	auth := &proxy.Auth{User: user, Password: passwd} // circuit isolation
	dialer, err := proxy.SOCKS5("tcp", proxyUrl.Hostname()+":"+proxyUrl.Port(), auth, new(net.Dialer))
	if err != nil {
		return nil, err
	}
	conn, err := dialer.Dial("tcp", serverUrl.Hostname()+":"+serverUrl.Port())
	if err != nil {
		return nil, err
	}
	if serverUrl.Scheme == "ircs" {
		cfg := &tls.Config{
			ServerName:         serverUrl.Hostname(),
			InsecureSkipVerify: !tlsVerify,
		}
		if clientAuthCert != nil {
			cfg.Certificates = append(cfg.Certificates, *clientAuthCert)
		}
		tlsConn := tls.Client(conn, cfg)
		if err = tlsConn.Handshake(); err != nil {
			return nil, err
		}
		conn = tlsConn
	}
	return conn, nil
}
