package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"plugin"

	"github.com/homedepot/ScriptPortal/endpoints/auth"
	"github.com/homedepot/ScriptPortal/endpoints/httpHelpers"
	"github.com/homedepot/ScriptPortal/endpoints/scriptrunner"
	"github.com/homedepot/ScriptPortal/globals"
)

var port int
var bind string
var pluginDirPath string

func init() {
	flag.IntVar(&port, "p", 80, "This is the port you wish to serve on")
	flag.StringVar(&bind, "bind", "0.0.0.0", "the address to bind to")
	flag.StringVar(&globals.ScriptConfigPath, "c", "/var/lib/ScriptPortal/scriptConfig.json", "the path to your configuration json file")
	flag.StringVar(&pluginDirPath, "plug", "/var/opt/ScriptPortal/", "the path of your compiled .so plugin files")
	flag.StringVar(&globals.TemplatePath, "template", "/usr/share/ScriptPortal/templates/", "the path of the go http template files")
	flag.Parse()

}

func generateHandleFunc(targetFunc func(http.ResponseWriter, *http.Request, chan []byte)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		retChannel := make(chan []byte, 1)
		targetFunc(w, r, retChannel)
		var toAddToPage []byte
		//if the channel has been closed before a write, assume the plugin handlefunc took care of it. otherwise, put the response in the normal page boilerplate.
		for x := range retChannel {
			toAddToPage = x
		}
		if len(toAddToPage) > 0 {
			httpHelpers.WrapInBoilerPlate(w, string(toAddToPage))
		}
		return

	}
}

func main() {
	authConfig, err := auth.LoadFromEnv()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	err = scriptrunner.Init(globals.ScriptConfigPath)
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	mux := http.NewServeMux()
	plugDirInfo, _ := ioutil.ReadDir(pluginDirPath)
	httpHelpers.Navs = append(httpHelpers.Navs, httpHelpers.TopBarInfo{Name: "Scripts", Link: "/"})
	var plugins []string
	for _, x := range plugDirInfo {
		plugins = append(plugins, x.Name())
	}
	for _, pluginPath := range plugins {
		plug, err := plugin.Open(pluginDirPath + "/" + pluginPath)
		if err != nil {
			fmt.Printf("error loading Plugin: %s, %s\n", pluginDirPath, err)
		}
		handlefunc, err := plug.Lookup("HandleRequest")
		if err != nil {
			fmt.Printf("error loading symbol HandleRequest: %s, %s\n", pluginDirPath, err)
		}
		NameFunc, err := plug.Lookup("Name")
		if err != nil {
			fmt.Printf("error loading symbol Name: %s, %s\n", pluginDirPath, err)
		}
		PathFunc, err := plug.Lookup("Path")
		if err != nil {
			fmt.Printf("error loading Symbol Path: %s, %s\n", pluginDirPath, err)
		}
		httpHelpers.Navs = append(httpHelpers.Navs, httpHelpers.TopBarInfo{
			Name: NameFunc.(func() string)(),
			Link: PathFunc.(func() string)(),
		})
		//Have fun parsing this one!
		mux.HandleFunc(PathFunc.(func() string)(), authConfig.RequireFunc(generateHandleFunc(handlefunc.(func(http.ResponseWriter, *http.Request, chan []byte)))))
	}
	mux.HandleFunc("/", authConfig.RequireFunc(scriptrunner.ScriptIndex))
	mux.HandleFunc("/script/", authConfig.RequireFunc(scriptrunner.ScriptMaster))
	mux.HandleFunc("/status/", authConfig.RequireFunc(scriptrunner.HandleSocketRequest))
	mux.HandleFunc("/search/", authConfig.RequireFunc(scriptrunner.ScriptSearch))
	fmt.Printf("Script Portal is running on %s:%d (authentication enabled: %t)\n", bind, port, authConfig.Enabled())
	if !authConfig.Enabled() {
		fmt.Println("WARNING: Script Portal is running without authentication.")
	}
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", bind, port), mux)
	if err != nil {
		fmt.Printf("it screwed up here: %s", err)
	}

}
