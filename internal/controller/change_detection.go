package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func (c *Controller) decideChange(etagHeader string, lastETag string, lastHash string, body []byte, lastETagIsHeader bool) bool {
	curHash := c.computeHash(body)
	if etagHeader != "" {
		return c.decideChangeWithETag(etagHeader, lastETag, lastHash, curHash, lastETagIsHeader)
	}
	return c.decideChangeNoETag(lastHash, curHash)
}

func (c *Controller) decideChangeWithETag(etagHeader, lastETag, lastHash, curHash string, lastETagIsHeader bool) bool {
	weak := c.isWeakETag(etagHeader)
	if !weak && lastETagIsHeader && etagHeader == lastETag {
		return false
	}
	if weak {
		if lastHash != "" && curHash == lastHash {
			return false
		}
		return true
	}
	if lastHash != "" && curHash == lastHash {
		return false
	}
	return true
}

func (*Controller) decideChangeNoETag(lastHash, curHash string) bool {
	if lastHash != "" && curHash == lastHash {
		return false
	}
	return true
}

func (*Controller) isWeakETag(et string) bool {
	return strings.HasPrefix(et, "W/")
}

func (*Controller) computeHash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
