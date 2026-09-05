import { fetchAPI } from './client';

export interface TLSCertificate {
    id: string;
    domain: string;
    issuer: string;
    subject: string;
    sans: string[];
    valid_from: string;
    valid_until: string;
    days_remaining: number;
    serial_number?: string;
    signature_algorithm?: string;
    ocsp_status?: string;
    last_verified_at: string;
    associated_endpoint_id?: string;
    service_name?: string;
}

export async function listCertificates(): Promise<TLSCertificate[]> {
    return fetchAPI<TLSCertificate[]>('/api/v1/certificates');
}

export async function testCertificate(host: string, port: number = 443): Promise<TLSCertificate> {
    return fetchAPI<TLSCertificate>('/api/v1/certificates/test', {
        method: 'POST',
        body: JSON.stringify({ host, port })
    });
}
