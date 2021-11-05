package plugindemo

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCopyRequestForWebhook(t *testing.T) {
	Convey("TestCopyRequestForWebhook", t, func() {
		var plugin Plugin
		plugin.webhook = "http://webhook.com"
		request, _ := http.NewRequest("POST", "http://test.com", strings.NewReader("testbody"))
		request.Header.Add("key1", "value1")
		request.Header.Add("key2", "value2")
		request.Header.Add("key1", "value3")
		request.Header.Add("key1", "value4")
		var cookie1 http.Cookie
		cookie1.Name = "Casbin-Plugin-ClientCode"
		cookie1.Value = "value"
		request.AddCookie(&cookie1)

		newRequest, err := plugin.copyRequestForWebhook(request)
		So(newRequest, ShouldNotBeNil)
		So(err, ShouldBeNil)
		body, err := ioutil.ReadAll(newRequest.Body)
		So(err, ShouldBeNil)
		So(string(body), ShouldEqual, "testbody")
		So(newRequest.URL.Host, ShouldEqual, "webhook.com")
		So(newRequest.Header, ShouldResemble, request.Header)
		cookie2, err := newRequest.Cookie("Casbin-Plugin-ClientCode")
		So(err, ShouldBeNil)
		So(cookie2, ShouldResemble, &cookie1)
	})
}
func TestModifyRequestForTraefik(t *testing.T) {
	Convey("TestModifyRequestForTraefik", t, func() {
		var plugin Plugin
		plugin.webhook = "http://webhook.com"
		request, _ := http.NewRequest("POST", "http://test.com", strings.NewReader("testbody"))
		request.Header.Add("key1", "value1")
		request.Header.Add("key2", "value2")
		request.Header.Add("key1", "value3")
		request.Header.Add("key1", "value4")
		var cookie1 http.Cookie
		cookie1.Name = "Casbin-Plugin-ClientCode"
		cookie1.Value = "value"
		request.AddCookie(&cookie1)

		var replacement Replacement
		replacement.ShouldReplaceBody = true
		replacement.ShouldReplaceHeader = true
		replacement.Body = "modified"
		replacement.Header = request.Header.Clone()
		delete(request.Header, "key2")
		replacement.Header["Cookie"] = []string{"Casbin-Plugin-ClientCode=value2"}

		newRequest, err := plugin.modifyRequestForTraefik(request, replacement)
		So(newRequest, ShouldNotBeNil)
		So(err, ShouldBeNil)
		body, err := ioutil.ReadAll(newRequest.Body)
		So(err, ShouldBeNil)
		So(string(body), ShouldEqual, "modified")
		So(newRequest.URL.Host, ShouldEqual, "webhook.com")
		So(map[string][]string(newRequest.Header), ShouldResemble, replacement.Header)
		cookie2, err := newRequest.Cookie("Casbin-Plugin-ClientCode")
		So(err, ShouldBeNil)
		So(cookie2.Value, ShouldResemble, "value2")
	})
}
