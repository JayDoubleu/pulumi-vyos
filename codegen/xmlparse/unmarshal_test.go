package xmlparse

import (
	"testing"
)

func TestParse_SystemHostName(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0"?>
<interfaceDefinition>
  <node name="system">
    <children>
      <leafNode name="host-name" owner="system_host-name.py">
        <properties>
          <help>System host name (default: vyos)</help>
        </properties>
      </leafNode>
    </children>
  </node>
</interfaceDefinition>`

	def, err := Parse([]byte(xml))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(def.Nodes) != 1 {
		t.Fatalf("expected 1 top-level node, got %d", len(def.Nodes))
	}
	if def.Nodes[0].Name != "system" {
		t.Errorf("node name = %q, want %q", def.Nodes[0].Name, "system")
	}
	if def.Nodes[0].Children == nil {
		t.Fatal("system node has no children")
	}
	if len(def.Nodes[0].Children.LeafNodes) != 1 {
		t.Fatalf("expected 1 leaf node, got %d", len(def.Nodes[0].Children.LeafNodes))
	}

	leaf := def.Nodes[0].Children.LeafNodes[0]
	if leaf.Name != "host-name" {
		t.Errorf("leaf name = %q, want %q", leaf.Name, "host-name")
	}
	if leaf.Owner != "system_host-name.py" {
		t.Errorf("leaf owner = %q, want %q", leaf.Owner, "system_host-name.py")
	}
	if leaf.Properties == nil || leaf.Properties.Help != "System host name (default: vyos)" {
		t.Errorf("leaf help = %q, want %q", leaf.Properties.Help, "System host name (default: vyos)")
	}
}

func TestParse_InterfacesEthernet(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0"?>
<interfaceDefinition>
  <node name="interfaces">
    <children>
      <tagNode name="ethernet" owner="interfaces_ethernet.py">
        <properties>
          <help>Ethernet Interface</help>
        </properties>
        <children>
          <leafNode name="description">
            <properties>
              <help>Description</help>
            </properties>
          </leafNode>
          <leafNode name="disable">
            <properties>
              <help>Disable interface</help>
              <valueless/>
            </properties>
          </leafNode>
          <leafNode name="mtu">
            <properties>
              <help>MTU</help>
              <valueHelp>
                <format>u32:68-16000</format>
                <description>MTU value</description>
              </valueHelp>
            </properties>
          </leafNode>
          <leafNode name="address">
            <properties>
              <help>IP address</help>
              <multi/>
            </properties>
          </leafNode>
          <node name="offload">
            <properties>
              <help>Configurable offload options</help>
            </properties>
            <children>
              <leafNode name="gro">
                <properties>
                  <help>Enable Generic Receive Offload</help>
                  <valueless/>
                </properties>
              </leafNode>
            </children>
          </node>
        </children>
      </tagNode>
    </children>
  </node>
</interfaceDefinition>`

	def, err := Parse([]byte(xml))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(def.Nodes) != 1 || def.Nodes[0].Name != "interfaces" {
		t.Fatal("expected interfaces node")
	}

	children := def.Nodes[0].Children
	if children == nil || len(children.TagNodes) != 1 {
		t.Fatal("expected 1 tagNode")
	}

	eth := children.TagNodes[0]
	if eth.Name != "ethernet" {
		t.Errorf("tagNode name = %q, want %q", eth.Name, "ethernet")
	}
	if eth.Owner != "interfaces_ethernet.py" {
		t.Errorf("owner = %q, want %q", eth.Owner, "interfaces_ethernet.py")
	}
	if eth.Children == nil {
		t.Fatal("ethernet has no children")
	}

	// Check leaf nodes
	leaves := eth.Children.LeafNodes
	if len(leaves) != 4 {
		t.Fatalf("expected 4 leaf nodes, got %d", len(leaves))
	}

	// Check valueless
	disable := findLeaf(leaves, "disable")
	if disable == nil {
		t.Fatal("disable leaf not found")
	}
	if disable.Properties.Valueless == nil {
		t.Error("disable should have valueless property")
	}

	// Check multi
	address := findLeaf(leaves, "address")
	if address == nil {
		t.Fatal("address leaf not found")
	}
	if address.Properties.Multi == nil {
		t.Error("address should have multi property")
	}

	// Check valueHelp (u32)
	mtu := findLeaf(leaves, "mtu")
	if mtu == nil {
		t.Fatal("mtu leaf not found")
	}
	if len(mtu.Properties.ValueHelps) != 1 || mtu.Properties.ValueHelps[0].Format != "u32:68-16000" {
		t.Errorf("mtu valueHelp format = %v, want u32:68-16000", mtu.Properties.ValueHelps)
	}

	// Check nested node
	if len(eth.Children.Nodes) != 1 || eth.Children.Nodes[0].Name != "offload" {
		t.Fatal("expected offload node")
	}
	offloadChildren := eth.Children.Nodes[0].Children
	if offloadChildren == nil || len(offloadChildren.LeafNodes) != 1 {
		t.Fatal("expected 1 leaf under offload")
	}
	if offloadChildren.LeafNodes[0].Name != "gro" {
		t.Errorf("offload leaf name = %q, want %q", offloadChildren.LeafNodes[0].Name, "gro")
	}
}

func TestParse_FirewallWithTagNodes(t *testing.T) {
	t.Parallel()

	xml := `<?xml version="1.0"?>
<interfaceDefinition>
  <node name="firewall" owner="firewall.py">
    <properties>
      <help>Firewall</help>
    </properties>
    <children>
      <tagNode name="flowtable">
        <properties>
          <help>Flowtable</help>
        </properties>
        <children>
          <leafNode name="offload">
            <properties>
              <help>Offloading method</help>
            </properties>
            <defaultValue>software</defaultValue>
          </leafNode>
        </children>
      </tagNode>
      <tagNode name="zone">
        <properties>
          <help>Zone-policy</help>
        </properties>
        <children>
          <leafNode name="default-action">
            <properties>
              <help>Default-action</help>
            </properties>
          </leafNode>
          <tagNode name="from">
            <properties>
              <help>Zone from which to filter traffic</help>
            </properties>
            <children>
              <node name="firewall">
                <children>
                  <leafNode name="name">
                    <properties>
                      <help>IPv4 firewall ruleset</help>
                    </properties>
                  </leafNode>
                </children>
              </node>
            </children>
          </tagNode>
        </children>
      </tagNode>
    </children>
  </node>
</interfaceDefinition>`

	def, err := Parse([]byte(xml))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	fw := def.Nodes[0]
	if fw.Name != "firewall" || fw.Owner != "firewall.py" {
		t.Fatalf("expected firewall node with owner, got %s (owner=%s)", fw.Name, fw.Owner)
	}
	if len(fw.Children.TagNodes) != 2 {
		t.Fatalf("expected 2 tagNodes under firewall, got %d", len(fw.Children.TagNodes))
	}

	// Check zone has nested tagNode "from"
	zone := findTagNode(fw.Children.TagNodes, "zone")
	if zone == nil {
		t.Fatal("zone tagNode not found")
	}
	if len(zone.Children.TagNodes) != 1 || zone.Children.TagNodes[0].Name != "from" {
		t.Fatal("expected 'from' tagNode under zone")
	}

	// Check defaultValue on flowtable > offload
	ft := findTagNode(fw.Children.TagNodes, "flowtable")
	if ft == nil {
		t.Fatal("flowtable tagNode not found")
	}
	offload := findLeaf(ft.Children.LeafNodes, "offload")
	if offload == nil || offload.DefaultValue == nil || *offload.DefaultValue != "software" {
		t.Error("offload defaultValue should be 'software'")
	}
}

func findLeaf(leaves []LeafNode, name string) *LeafNode {
	for i := range leaves {
		if leaves[i].Name == name {
			return &leaves[i]
		}
	}
	return nil
}

func findTagNode(tags []TagNode, name string) *TagNode {
	for i := range tags {
		if tags[i].Name == name {
			return &tags[i]
		}
	}
	return nil
}
