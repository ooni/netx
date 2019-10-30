// Package logger is a handler that emits logs
package logger

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/apex/log"
	"github.com/ooni/netx/model"
)

var (
	tlsVersion = map[uint16]string{
		tls.VersionSSL30: "SSLv3",
		tls.VersionTLS10: "TLSv1",
		tls.VersionTLS11: "TLSv1.1",
		tls.VersionTLS12: "TLSv1.2",
		tls.VersionTLS13: "TLSv1.3",
	}
)

// Handler is a handler that logs events.
type Handler struct {
	logger log.Interface
}

// NewHandler returns a new logging handler.
func NewHandler(logger log.Interface) *Handler {
	return &Handler{logger: logger}
}

// OnMeasurement logs the specific measurement
func (h *Handler) OnMeasurement(m model.Measurement) {
	// DNS
	if m.ResolveStart != nil {
		h.logger.WithFields(log.Fields{
			"dialID":        m.ResolveStart.DialID,
			"elapsed":       m.ResolveStart.DurationSinceBeginning,
			"hostname":      m.ResolveStart.Hostname,
			"transactionID": m.ResolveStart.TransactionID,
		}).Debug("dns: resolve domain name")
	}
	if m.DNSQuery != nil {
		h.logger.WithFields(log.Fields{
			"dialID":   m.DNSQuery.DialID,
			"elapsed":  m.DNSQuery.DurationSinceBeginning,
			"numBytes": len(m.DNSQuery.Data),
			"value":    fmt.Sprintf("\n\n\t%s", reformat(m.DNSQuery.Msg.String())),
		}).Debug("dns: query out")
	}
	if m.DNSReply != nil {
		h.logger.WithFields(log.Fields{
			"dialID":   m.DNSReply.DialID,
			"elapsed":  m.DNSReply.DurationSinceBeginning,
			"numBytes": len(m.DNSReply.Data),
			"value":    fmt.Sprintf("\n\n\t%s", reformat(m.DNSReply.Msg.String())),
		}).Debug("dns: reply in")
	}
	if m.ResolveDone != nil {
		h.logger.WithFields(log.Fields{
			"addresses":     m.ResolveDone.Addresses,
			"dialID":        m.ResolveDone.DialID,
			"elapsed":       m.ResolveDone.DurationSinceBeginning,
			"error":         m.ResolveDone.Error,
			"transactionID": m.ResolveDone.TransactionID,
		}).Debug("dns: resolution done")
	}

	// Syscalls
	if m.Connect != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor":    m.Connect.SyscallDuration,
			"connID":        m.Connect.ConnID,
			"dialID":        m.Connect.DialID,
			"elapsed":       m.Connect.DurationSinceBeginning,
			"error":         m.Connect.Error,
			"network":       m.Connect.Network,
			"remoteAddress": m.Connect.RemoteAddress,
			"transactionID": m.Connect.TransactionID,
		}).Debug("net: connect done")
	}
	if m.Read != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor": m.Read.SyscallDuration,
			"connID":     m.Read.ConnID,
			"elapsed":    m.Read.DurationSinceBeginning,
			"error":      m.Read.Error,
			"numBytes":   m.Read.NumBytes,
		}).Debug("net: read done")
	}
	if m.Write != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor": m.Write.SyscallDuration,
			"connID":     m.Write.ConnID,
			"elapsed":    m.Write.DurationSinceBeginning,
			"error":      m.Write.Error,
			"numBytes":   m.Write.NumBytes,
		}).Debug("net: write done")
	}
	if m.Close != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor": m.Close.SyscallDuration,
			"connID":     m.Close.ConnID,
			"elapsed":    m.Close.DurationSinceBeginning,
		}).Debug("net: close done")
	}

	// TLS
	if m.TLSHandshakeStart != nil {
		h.logger.WithFields(log.Fields{
			"connID":        m.TLSHandshakeStart.ConnID,
			"elapsed":       m.TLSHandshakeStart.DurationSinceBeginning,
			"transactionID": m.TLSHandshakeStart.TransactionID,
		}).Debug("tls: start handshake")
	}
	if m.TLSHandshakeDone != nil {
		h.logger.WithFields(log.Fields{
			"alpn":          m.TLSHandshakeDone.ConnectionState.NegotiatedProtocol,
			"connID":        m.TLSHandshakeDone.ConnID,
			"elapsed":       m.TLSHandshakeDone.DurationSinceBeginning,
			"error":         m.TLSHandshakeDone.Error,
			"transactionID": m.TLSHandshakeDone.TransactionID,
			"version":       tlsVersion[m.TLSHandshakeDone.ConnectionState.Version],
		}).Debug("tls: handshake done")
	}

	// HTTP round trip
	if m.HTTPRoundTripStart != nil {
		h.logger.WithFields(log.Fields{
			"dialID":        m.HTTPRoundTripStart.DialID,
			"elapsed":       m.HTTPRoundTripStart.DurationSinceBeginning,
			"method":        m.HTTPRoundTripStart.Method,
			"transactionID": m.HTTPRoundTripStart.TransactionID,
			"url":           m.HTTPRoundTripStart.URL,
		}).Debug("http: start round trip")
	}
	if m.HTTPConnectionReady != nil {
		h.logger.WithFields(log.Fields{
			"connID":        m.HTTPConnectionReady.ConnID,
			"elapsed":       m.HTTPConnectionReady.DurationSinceBeginning,
			"transactionID": m.HTTPConnectionReady.TransactionID,
		}).Debug("http: connection ready")
	}
	if m.HTTPRequestHeader != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRequestHeader.DurationSinceBeginning,
			"key":           m.HTTPRequestHeader.Key,
			"transactionID": m.HTTPRequestHeader.TransactionID,
			"value":         m.HTTPRequestHeader.Value,
		}).Debug("http: header out")
	}
	if m.HTTPRequestHeadersDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRequestHeadersDone.DurationSinceBeginning,
			"transactionID": m.HTTPRequestHeadersDone.TransactionID,
		}).Debug("http: all headers out")
	}
	if m.HTTPRequestDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRequestDone.DurationSinceBeginning,
			"transactionID": m.HTTPRequestDone.TransactionID,
		}).Debug("http: whole request out")
	}
	if m.HTTPResponseStart != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPResponseStart.DurationSinceBeginning,
			"transactionID": m.HTTPResponseStart.TransactionID,
		}).Debug("http: first response byte")
	}
	if m.HTTPRoundTripDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":         m.HTTPRoundTripDone.DurationSinceBeginning,
			"error":           m.HTTPRoundTripDone.Error,
			"headers":         m.HTTPRoundTripDone.Headers,
			"redirect_body":   stringifyBody(m.HTTPRoundTripDone.RedirectBody),
			"request_method":  m.HTTPRoundTripDone.RequestMethod,
			"request_headers": m.HTTPRoundTripDone.RequestHeaders,
			"request_url":     m.HTTPRoundTripDone.RequestURL,
			"statusCode":      m.HTTPRoundTripDone.StatusCode,
			"transactionID":   m.HTTPRoundTripDone.TransactionID,
		}).Debug("http: round trip done")
	}

	// HTTP response body
	if m.HTTPResponseBodyPart != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPResponseBodyPart.DurationSinceBeginning,
			"error":         m.HTTPResponseBodyPart.Error,
			"numBytes":      len(m.HTTPResponseBodyPart.Data),
			"transactionID": m.HTTPResponseBodyPart.TransactionID,
		}).Debug("http: got body part")
	}
	if m.HTTPResponseDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPResponseDone.DurationSinceBeginning,
			"transactionID": m.HTTPResponseDone.TransactionID,
		}).Debug("http: got whole body")
	}

	// Extensions
	if m.Extension != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.Extension.DurationSinceBeginning,
			"key":           m.Extension.Key,
			"severity":      m.Extension.Severity,
			"transactionID": m.Extension.TransactionID,
			"value":         fmt.Sprintf("%+v", m.Extension.Value),
		}).Debug("extension:")
	}
}

func reformat(s string) string {
	return strings.ReplaceAll(s, "\n", "\n\t")
}

func stringifyBody(d []byte) string {
	return string(bytes.ReplaceAll(d, []byte("\n"), []byte(`\n`)))
}
