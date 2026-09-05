import type { Component } from "svelte";
import DnsCheckCard from "./DnsCheckCard.svelte";
import GenericCheckCard from "./GenericCheckCard.svelte";
import HttpCheckCard from "./HttpCheckCard.svelte";
import PingCheckCard from "./PingCheckCard.svelte";
import PushCheckCard from "./PushCheckCard.svelte";
import TcpCheckCard from "./TcpCheckCard.svelte";
import type { CheckCardProps } from "./_shared/types";

export type CheckCardComponent = Component<CheckCardProps>;

const registry: Record<string, CheckCardComponent> = {
    dns: DnsCheckCard,
    http: HttpCheckCard,
    https: HttpCheckCard,
    ping: PingCheckCard,
    push: PushCheckCard,
    tcp: TcpCheckCard,
};

export function cardFor(type: string): CheckCardComponent {
    return registry[type] ?? GenericCheckCard;
}

export { GenericCheckCard };
