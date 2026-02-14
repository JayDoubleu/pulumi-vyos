import * as vyos from "@jaydoubleu/vyos";

const hostname = new vyos.SystemHostName("hostname", {
    hostName: "my-vyos-router",
});

export const hostnameValue = hostname.hostName;
