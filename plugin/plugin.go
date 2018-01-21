package plugin

import (
	"errors"
	"os/exec"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/michelvocks/gaia/proto"
)

const (
	pluginMapKey = "plugin"
)

var handshake = plugin.HandshakeConfig{
	ProtocolVersion: 1,
	MagicCookieKey:  "GAIA_PLUGIN",
	// This cookie should never be changed again
	MagicCookieValue: "FdXjW27mN6XuG2zDBP4LixXUwDAGCEkidxwqBGYpUhxiWHzctATYZvpz4ZJdALmh",
}

var pluginMap = map[string]plugin.Plugin{
	pluginMapKey: &PluginGRPCImpl{},
}

// Plugin represents a single plugin instance which uses gRPC
// to connect to exactly one plugin.
type Plugin struct {
	// Client instance used to open gRPC connections.
	client *plugin.Client

	// Interface to the connected plugin.
	pluginConn PluginGRPC
}

// NewPlugin creates a new instance of Plugin.
// One Plugin instance represents one connection to a plugin.
func NewPlugin(c *exec.Cmd) *Plugin {
	// Allocate
	p := &Plugin{}

	// Get new client
	p.client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  handshake,
		Plugins:          pluginMap,
		Cmd:              c,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	return p
}

// Connect starts the plugin, initiates the gRPC connection and looks up the plugin.
// It's up to the caller to call plugin.Close to shutdown the plugin
// and close the gRPC connection.
func (p *Plugin) Connect() error {
	// Connect via gRPC
	gRPCClient, err := p.client.Client()
	if err != nil {
		return err
	}

	// Request the plugin
	raw, err := gRPCClient.Dispense(pluginMapKey)
	if err != nil {
		return err
	}

	// Convert plugin to interface
	if pC, ok := raw.(PluginGRPC); ok {
		p.pluginConn = pC
		return nil
	}

	return errors.New("plugin is not compatible with Gaia plugin interface")
}

// Execute triggers the execution of one single job
// for the given plugin.
func (p *Plugin) Execute(j *proto.Job) error {
	_, err := p.pluginConn.ExecuteJob(j)
	return err
}

// Close shutdown the plugin and kills the gRPC connection.
// Remember to call this when you call plugin.Connect.
func (p *Plugin) Close() {
	// We start the kill command in a goroutine because kill
	// is blocking until the subprocess successfully exits.
	// The user should not notice the stopping process.
	go func() {
		p.client.Kill()
	}()
}