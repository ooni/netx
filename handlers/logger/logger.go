// Package logger is a handler that emits logs
package logger

import (
	"crypto/tls"

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
			"dialID":   m.ResolveStart.DialID,
			"elapsed":  m.ResolveStart.Time,
			"hostname": m.ResolveStart.Hostname,
		}).Debug("dns: resolve domain name")
	}
	if m.ResolveDone != nil {
		h.logger.WithFields(log.Fields{
			"addresses": m.ResolveDone.Addresses,
			"dialID":    m.ResolveDone.DialID,
			"elapsed":   m.ResolveDone.Time,
			"error":     m.ResolveDone.Error,
		}).Debug("dns: resolution done")
	}

	// Syscalls
	if m.Connect != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor":    m.Connect.Duration,
			"connID":        m.Connect.ConnID,
			"dialID":        m.Connect.DialID,
			"elapsed":       m.Connect.Time,
			"error":         m.Connect.Error,
			"network":       m.Connect.Network,
			"remoteAddress": m.Connect.RemoteAddress,
		}).Debug("net: connect done")
	}
	if m.Read != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor": m.Read.Duration,
			"connID":     m.Read.ConnID,
			"elapsed":    m.Read.Time,
			"numBytes":   m.Read.NumBytes,
		}).Debug("net: read done")
	}
	if m.Write != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor": m.Write.Duration,
			"connID":     m.Write.ConnID,
			"elapsed":    m.Write.Time,
			"numBytes":   m.Write.NumBytes,
		}).Debug("net: write done")
	}
	if m.Close != nil {
		h.logger.WithFields(log.Fields{
			"blockedFor": m.Close.Duration,
			"connID":     m.Close.ConnID,
			"elapsed":    m.Close.Time,
		}).Debug("net: close done")
	}

	// TLS
	if m.TLSHandshakeStart != nil {
		h.logger.WithFields(log.Fields{
			"connID":        m.TLSHandshakeStart.ConnID,
			"elapsed":       m.TLSHandshakeStart.Time,
			"transactionID": m.TLSHandshakeStart.TransactionID,
		}).Debug("tls: start handshake")
	}
	if m.TLSHandshakeDone != nil {
		h.logger.WithFields(log.Fields{
			"alpn":          m.TLSHandshakeDone.ConnectionState.NegotiatedProtocol,
			"connID":        m.TLSHandshakeDone.ConnID,
			"elapsed":       m.TLSHandshakeDone.Time,
			"error":         m.TLSHandshakeDone.Error,
			"transactionID": m.TLSHandshakeDone.TransactionID,
			"version":       tlsVersion[m.TLSHandshakeDone.ConnectionState.Version],
		}).Debug("tls: handshake done")
	}

	// HTTP round trip
	if m.HTTPRoundTripStart != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRoundTripStart.Time,
			"method":        m.HTTPRoundTripStart.Method,
			"transactionID": m.HTTPRoundTripStart.TransactionID,
			"url":           m.HTTPRoundTripStart.URL,
		}).Debug("http: start round trip")
	}
	if m.HTTPConnectionReady != nil {
		h.logger.WithFields(log.Fields{
			"connID":        m.HTTPConnectionReady.ConnID,
			"elapsed":       m.HTTPConnectionReady.Time,
			"transactionID": m.HTTPConnectionReady.TransactionID,
		}).Debug("http: connection ready")
	}
	if m.HTTPRequestHeader != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRequestHeader.Time,
			"key":           m.HTTPRequestHeader.Key,
			"transactionID": m.HTTPRequestHeader.TransactionID,
			"value":         m.HTTPRequestHeader.Value,
		}).Debug("http: header out")
	}
	if m.HTTPRequestHeadersDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRequestHeadersDone.Time,
			"transactionID": m.HTTPRequestHeadersDone.TransactionID,
		}).Debug("http: all headers out")
	}
	if m.HTTPRequestDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRequestDone.Time,
			"transactionID": m.HTTPRequestDone.TransactionID,
		}).Debug("http: whole request out")
	}
	if m.HTTPResponseStart != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPResponseStart.Time,
			"transactionID": m.HTTPResponseStart.TransactionID,
		}).Debug("http: first response byte")
	}
	if m.HTTPRoundTripDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPRoundTripDone.Time,
			"error":         m.HTTPRoundTripDone.Error,
			"statusCode":    m.HTTPRoundTripDone.StatusCode,
			"transactionID": m.HTTPRoundTripDone.TransactionID,
		}).Debug("http: round trip done")
		for key, values := range m.HTTPRoundTripDone.Headers {
			for _, value := range values {
				h.logger.WithFields(log.Fields{
					"elapsed":       m.HTTPRoundTripDone.Time,
					"key":           key,
					"transactionID": m.HTTPRoundTripDone.TransactionID,
					"value":         value,
				}).Debug("http: got header")
			}
		}
	}

	// HTTP response body
	if m.HTTPResponseBodyPart != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPResponseBodyPart.Time,
			"error":         m.HTTPResponseBodyPart.Error,
			"numBytes":      m.HTTPResponseBodyPart.NumBytes,
			"transactionID": m.HTTPResponseBodyPart.TransactionID,
		}).Debug("http: got body part")
	}
	if m.HTTPResponseDone != nil {
		h.logger.WithFields(log.Fields{
			"elapsed":       m.HTTPResponseDone.Time,
			"transactionID": m.HTTPResponseDone.TransactionID,
		}).Debug("http: got whole body")
	}
}
