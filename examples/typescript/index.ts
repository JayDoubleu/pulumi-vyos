import * as vyos from "@jaydoubleu/pulumi-vyos";

const hostname = new vyos.SystemHostname("hostname", {
    hostname: "my-vyos-router",
});

export const hostnameValue = hostname.hostname;
