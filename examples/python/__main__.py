import pulumi
import pulumi_vyos as vyos

hostname = vyos.SystemHostname("hostname", hostname="my-vyos-router")

pulumi.export("hostname", hostname.hostname)
