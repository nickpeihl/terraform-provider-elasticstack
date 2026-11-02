// PROTOTYPE (spike G, throwaway): protocol-level override of GenerateResourceConfig for
// elasticstack_kibana_dashboard. plugin-framework v1.19 has no per-resource hook, so we wrap
// the tfprotov6 server and post-process the framework's state->config value.
//
// Fix applied: vis/markdown panels' Read populates BOTH the typed block (vis_config /
// markdown_config) AND the Optional+Computed panel-level config_json mirror. The generated
// config then violates ConflictsWith. We null config_json whenever a typed *_config sibling is set.

package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const protoGDashboardType = "elasticstack_kibana_dashboard"

type protoGGenConfigServer struct {
	tfprotov6.ProviderServer
}

func protoGWrap(f func() tfprotov6.ProviderServer) func() tfprotov6.ProviderServer {
	return func() tfprotov6.ProviderServer { return protoGGenConfigServer{f()} }
}

func (s protoGGenConfigServer) GenerateResourceConfig(ctx context.Context, req *tfprotov6.GenerateResourceConfigRequest) (*tfprotov6.GenerateResourceConfigResponse, error) {
	resp, err := s.ProviderServer.GenerateResourceConfig(ctx, req)
	if err != nil || resp == nil || resp.Config == nil || req.TypeName != protoGDashboardType || os.Getenv("PROTO_G_GENCONFIG") == "off" {
		return resp, err
	}
	sch, err := s.ProviderServer.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		return resp, err
	}
	ty := sch.ResourceSchemas[protoGDashboardType].ValueType()
	val, err := resp.Config.Unmarshal(ty)
	if err != nil {
		return resp, err
	}
	if os.Getenv("PROTO_G_STRIP") == "off" {
		return resp, nil
	}
	out, err := tftypes.Transform(val, protoGStripConfigJSONMirror)
	if err != nil {
		return resp, err
	}
	dv, err := tfprotov6.NewDynamicValue(ty, out)
	if err != nil {
		return resp, err
	}
	resp.Config = &dv
	return resp, nil
}

// protoGStripConfigJSONMirror nulls config_json on any panel object that also has a typed *_config set.
func protoGStripConfigJSONMirror(_ *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
	objTy, ok := v.Type().(tftypes.Object)
	if !ok || v.IsNull() || !v.IsKnown() {
		return v, nil
	}
	if _, hasCfg := objTy.AttributeTypes["config_json"]; !hasCfg {
		return v, nil
	}
	if _, hasGrid := objTy.AttributeTypes["grid"]; !hasGrid {
		return v, nil
	}
	attrs := map[string]tftypes.Value{}
	if err := v.As(&attrs); err != nil {
		return v, err
	}
	typed := false
	for k, a := range attrs {
		if strings.HasSuffix(k, "_config") && !a.IsNull() {
			typed = true
		}
	}
	if !typed || attrs["config_json"].IsNull() {
		return v, nil
	}
	attrs["config_json"] = tftypes.NewValue(objTy.AttributeTypes["config_json"], nil)
	return tftypes.NewValue(objTy, attrs), nil
}

// plugin-framework v1.19.0 sets fwserver.ServerCapabilities.GenerateResourceConfig=true but
// internal/toproto6/server_capabilities.go drops it, so Terraform core never calls the RPC and
// uses its own legacy generator. Announce the capability ourselves.
func (s protoGGenConfigServer) GetProviderSchema(ctx context.Context, req *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	resp, err := s.ProviderServer.GetProviderSchema(ctx, req)
	if resp != nil && resp.ServerCapabilities != nil && os.Getenv("PROTO_G_GENCONFIG") != "off" {
		resp.ServerCapabilities.GenerateResourceConfig = true
	}
	return resp, err
}

func (s protoGGenConfigServer) GetMetadata(ctx context.Context, req *tfprotov6.GetMetadataRequest) (*tfprotov6.GetMetadataResponse, error) {
	resp, err := s.ProviderServer.GetMetadata(ctx, req)
	if resp != nil && resp.ServerCapabilities != nil && os.Getenv("PROTO_G_GENCONFIG") != "off" {
		resp.ServerCapabilities.GenerateResourceConfig = true
	}
	return resp, err
}
