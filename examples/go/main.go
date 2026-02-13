package main

import (
	"github.com/jaydoubleu/pulumi-vyos/sdk/go/vyos"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		hostname, err := vyos.NewSystemHostname(ctx, "hostname", &vyos.SystemHostnameArgs{
			Hostname: pulumi.String("my-vyos-router"),
		})
		if err != nil {
			return err
		}

		ctx.Export("hostname", hostname.Hostname)
		return nil
	})
}
