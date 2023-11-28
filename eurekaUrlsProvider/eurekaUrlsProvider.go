package eurekaurlsprovider

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Applications struct {
	XMLName      xml.Name      `xml:"applications"`
	Applications []Application `xml:"application"`
}
type Application struct {
	XMLName  xml.Name `xml:"application"`
	Name     string   `xml:"name"`
	Instance Instance `xml:"instance"`
}
type Instance struct {
	XMLName    xml.Name   `xml:"instance"`
	App        string     `xml:"app"`
	HostName   string     `xml:"hostName"`
	IpAddr     string     `xml:"ipAddr"`
	Status     string     `xml:"status"`
	Port       Port       `xml:"port"`
	SecurePort SecurePort `xml:"securePort"`
}

type Port struct {
	XMLName   xml.Name `xml:"port"`
	Enabled   bool     `xml:"enabled,attr"`
	PortValue string   `xml:",innerxml"`
}

type SecurePort struct {
	XMLName   xml.Name `xml:"securePort"`
	Enabled   bool     `xml:"enabled,attr"`
	PortValue string   `xml:",innerxml"`
}

func LoadEurekaURLs(urldest string, eurekaService string, eurekaExtraUri string) []*url.URL {
	var urls []*url.URL
	resp, err := http.Get(urldest)

	if err != nil {
		panic(err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var applications Applications
	err = xml.Unmarshal(b, &applications)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	for i := 0; i < len(applications.Applications); i++ {
		if strings.Contains(applications.Applications[i].Name, eurekaService) {
			rawUrl := applications.Applications[i].Instance.HostName
			if !strings.HasPrefix(rawUrl, "http") {
				if applications.Applications[i].Instance.Port.Enabled {
					rawUrl = "http://" + rawUrl
				} else if applications.Applications[i].Instance.SecurePort.Enabled {
					rawUrl = "https://" + rawUrl
				}
			}

			if applications.Applications[i].Instance.Port.Enabled {
				rawUrl = rawUrl + ":" + applications.Applications[i].Instance.Port.PortValue
			} else if applications.Applications[i].Instance.SecurePort.Enabled {
				rawUrl = rawUrl + ":" + applications.Applications[i].Instance.SecurePort.PortValue
			}

			URL, err := url.Parse(rawUrl + eurekaExtraUri)
			if err != nil {
				panic(err)
			}

			urls = append(urls, URL)
		}
	}

	return urls
}
