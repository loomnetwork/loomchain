package plugin

import "testing"

func TestPluginSoLoader(t *testing.T) {
	e := NewExternalLoader("")

	//TODO this test would be more inclusive if I created the plugin files

	_, err1 := e.LoadContract("test")

	if err1 != ErrPluginNotFound {
		t.Errorf("wrong error for test, we got %s, wanted ErrPluginNotFound", err1.Error())
	}

	_, err2 := e.LoadContract("test.so.1.0.0")
	if err2 != ErrPluginNotFound {
		t.Errorf("wrong error for test.so.1.0.0, we got %s, wanted ErrPluginNotFound", err2.Error())
	}

}

func TestWASMContractClient_Meta(t *testing.T) {
	client := &WASMContractClient{
		cmd:"../../life/cmd/gowasm/gowasm",
		path:"../../weave-blueprint/src/wasm/main.wasm",
	}
	meta, err := client.Meta()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%#v", meta)
}