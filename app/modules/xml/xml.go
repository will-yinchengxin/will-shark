package xml

import "encoding/xml"

type ManifestInterface interface {
	GetDevice() string
}

type ManifestWhole struct {
	XMLName       xml.Name `json:"xmlName" xml:"manifest"`
	Text          string   `json:"text" xml:",chardata"`
	Version       string   `json:"version" xml:"version,attr"`
	VersionNumber string   `json:"versionNumber" xml:"versionNumber,attr"`
	Device        string   `json:"device" xml:"device,attr"`
}

func (m *ManifestWhole) GetDevice() string {
	return m.Device
}

type ManifestDiff struct {
	XMLName       xml.Name `json:"xmlName" xml:"manifest"`
	Text          string   `json:"text" xml:",chardata"`
	Upgraded      string   `json:"upgraded" xml:"upgraded,attr"`
	UpgradeNumber string   `json:"upgradeNumber" xml:"upgradeNumber,attr"`
	Version       string   `json:"version" xml:"version,attr"`
	VersionNumber string   `json:"versionNumber" xml:"versionNumber,attr"`
	Device        string   `json:"device" xml:"device,attr"`
}

func (m *ManifestDiff) GetDevice() string {
	return m.Device
}

type NordicFile struct {
	XMLName    xml.Name `json:"xmlName" xml:"manifest"`
	Text       string   `json:"text" xml:",chardata"`
	Version    string   `json:"version" xml:"version,attr"`
	DeviceType string   `json:"deviceType" xml:"deviceType,attr"`
	Device     string   `json:"device" xml:"device,attr"`
}

func (m *NordicFile) GetDevice() string {
	return m.Device
}
