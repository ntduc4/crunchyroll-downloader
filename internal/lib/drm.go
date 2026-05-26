package lib

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/iyear/gowidevine"
	"github.com/iyear/gowidevine/widevinepb"
	"github.com/unki2aut/go-mpd"
)

var keys []*widevine.Key

// getPssh finds the first PSSH anywhere in the MPD manifest.
func getPssh(mpd *mpd.MPD) *string {
	for _, period := range mpd.Period {
		for _, set := range period.AdaptationSets {
			if pssh := getPsshFromProtections(set.ContentProtections); pssh != nil {
				return pssh
			}

			for _, representation := range set.Representations {
				if pssh := getPsshFromProtections(representation.ContentProtections); pssh != nil {
					return pssh
				}
			}
		}
	}

	return nil
}

func getPsshFromProtections(contentProtections []mpd.Descriptor) *string {
	for _, contentProtection := range contentProtections {
		if contentProtection.CencPSSH != nil {
			return contentProtection.CencPSSH
		}
	}

	return nil
}

func getDefaultKID(mpd *mpd.MPD) *string {
	for _, period := range mpd.Period {
		for _, set := range period.AdaptationSets {
			if kid := getDefaultKIDFromProtections(set.ContentProtections); kid != nil {
				return kid
			}
			for _, representation := range set.Representations {
				if kid := getDefaultKIDFromProtections(representation.ContentProtections); kid != nil {
					return kid
				}
			}
		}
	}

	return nil
}

func getDefaultKIDFromProtections(contentProtections []mpd.Descriptor) *string {
	for _, contentProtection := range contentProtections {
		if contentProtection.CencDefaultKeyId != nil {
			return contentProtection.CencDefaultKeyId
		}
	}

	return nil
}

type CrunchyrollWidevineLicenseResponse struct {
	License string `json:"license"`
}

func sendChallenge(contentId, videoToken string, challenge []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, "https://www.crunchyroll.com/license/v1/license/widevine", io.NopCloser(bytes.NewReader(challenge)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Cr-Content-Id", contentId)
	req.Header.Set("X-Cr-Video-Token", videoToken)
	req.Header.Set("Authorization", "Bearer "+Token)
	req.Header.Set("Origin", "https://static.crunchyroll.com")
	req.Header.Set("Referer", "https://static.crunchyroll.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:147.0) Gecko/20100101 Firefox/147.0")
	resp, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse JSON response
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result CrunchyrollWidevineLicenseResponse
	if err = json.Unmarshal(res, &result); err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(result.License)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func getWidevineDevice() (*widevine.Device, error) {
	var clientID []byte
	var privateKey []byte
	files, _ := os.ReadDir(".")
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".wvd") {
			wvd, err := os.Open(file.Name())
			if err != nil {
				return nil, err
			}

			return widevine.NewDevice(widevine.FromWVD(io.NopCloser(wvd)))
		} else if file.Name() == "client_id.bin" {
			f, err := os.Open("client_id.bin")
			if err != nil {
				return nil, err
			}
			defer f.Close()

			clientID, err = io.ReadAll(f)
		} else if file.Name() == "private_key.pem" {
			f, err := os.Open("private_key.pem")
			if err != nil {
				return nil, err
			}
			defer f.Close()

			privateKey, err = io.ReadAll(f)
			break
		}
	}

	if len(clientID) > 0 && len(privateKey) > 0 {
		return widevine.NewDevice(widevine.FromRaw(clientID, privateKey))
	}

	return nil, nil
}

func GetLicense(psshData, contentId, videoToken string, defaultKID *string) error {
	device, err := getWidevineDevice()
	if device == nil {
		return errors.New("no widevine device provided. You either need:\n- a \".wvd\" file,\n- or \"client_id.bin\" and \"private_key.pem\" files.\nI'm not sharing links for obvious reasons, but search \"ready to use cdms\" on Google :)\n")
	} else if err != nil {
		return err
	}
	cdm := widevine.NewCDM(device)
	decodedPssh, err := base64.StdEncoding.DecodeString(psshData)
	if err != nil {
		return err
	}
	pssh, err := widevine.NewPSSH(decodedPssh)
	if err != nil {
		return err
	}

	challenge, parseLicense, err := cdm.GetLicenseChallenge(pssh, widevinepb.LicenseType_AUTOMATIC, false)
	if err != nil {
		return err
	}
	resp, err := sendChallenge(contentId, videoToken, challenge)
	if err != nil {
		return err
	}
	keys, err = parseLicense(resp)
	if err != nil {
		return err
	}
	if defaultKID != nil {
		keys, err = selectLicenseKey(keys, *defaultKID)
		if err != nil {
			return err
		}
	}

	return nil
}

func selectLicenseKey(licenseKeys []*widevine.Key, defaultKID string) ([]*widevine.Key, error) {
	kidBytes, err := hex.DecodeString(strings.ReplaceAll(defaultKID, "-", ""))
	if err != nil {
		return nil, fmt.Errorf("invalid default_KID %q: %w", defaultKID, err)
	}

	for _, key := range licenseKeys {
		if key.Type == widevinepb.License_KeyContainer_CONTENT && bytes.Equal(key.ID, kidBytes) {
			return []*widevine.Key{key}, nil
		}
	}

	return nil, fmt.Errorf("no license key matches default_KID %s", defaultKID)
}
