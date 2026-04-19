import type { Component } from "svelte";
import CompositeCheckCard from "./CompositeCheckCard.svelte";
import DbCheckCard from "./DbCheckCard.svelte";
import DnsCheckCard from "./DnsCheckCard.svelte";
import DnsHttpCheckCard from "./DnsHttpCheckCard.svelte";
import GenericCheckCard from "./GenericCheckCard.svelte";
import HttpCheckCard from "./HttpCheckCard.svelte";
import HttpsCheckCard from "./HttpsCheckCard.svelte";
import JsonCheckCard from "./JsonCheckCard.svelte";
import PingCheckCard from "./PingCheckCard.svelte";
import PushCheckCard from "./PushCheckCard.svelte";
import SmtpCheckCard from "./SmtpCheckCard.svelte";
import SshCheckCard from "./SshCheckCard.svelte";
import SslCheckCard from "./SslCheckCard.svelte";
import TcpCheckCard from "./TcpCheckCard.svelte";
import TransactionCheckCard from "./TransactionCheckCard.svelte";
import UdpCheckCard from "./UdpCheckCard.svelte";
import WebsocketCheckCard from "./WebsocketCheckCard.svelte";
import type { CheckCardProps } from "./_shared/types";

export type CheckCardComponent = Component<CheckCardProps>;

const registry: Record<string, CheckCardComponent> = {
    composite: CompositeCheckCard,
    dns: DnsCheckCard,
    dns_http: DnsHttpCheckCard,
    http: HttpCheckCard,
    https: HttpsCheckCard,
    json: JsonCheckCard,
    mongo: DbCheckCard,
    mysql: DbCheckCard,
    ping: PingCheckCard,
    postgres: DbCheckCard,
    push: PushCheckCard,
    redis: DbCheckCard,
    smtp: SmtpCheckCard,
    ssh: SshCheckCard,
    ssl: SslCheckCard,
    tcp: TcpCheckCard,
    transaction: TransactionCheckCard,
    udp: UdpCheckCard,
    websocket: WebsocketCheckCard,
};

export function cardFor(type: string): CheckCardComponent {
    return registry[type] ?? GenericCheckCard;
}

export { GenericCheckCard };
