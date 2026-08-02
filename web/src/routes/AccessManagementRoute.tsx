import { useEffect, useState } from 'react';
import { useAuth, useTenant, getAccessToken } from '../auth/session';
import { hasPlatformAdmin } from '../auth/claims';
import { useAppProfiles } from '../api/queries';

type Grant = { subject: string; display_name: string; email: string; app_id: string; permission: string };
type DirectoryUser = { subject: string; username: string; display_name: string; email: string; enabled: boolean };
const headers = () => ({ 'Content-Type': 'application/json', ...(getAccessToken() ? { Authorization: `Bearer ${getAccessToken()}` } : {}) });

export function AccessManagementRoute() {
  const { claims, authEnabled } = useAuth();
  const tenant = useTenant();
  const [grants, setGrants] = useState<Grant[]>([]);
  const [error, setError] = useState('');
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<DirectoryUser[]>([]);
  const [searching, setSearching] = useState(false);
  const profileQuery = useAppProfiles();
  const grantableProfiles = (profileQuery.data?.app_profiles ?? []).filter((profile) => profile.app_id !== 'console');
  const [form, setForm] = useState({ subject: '', display_name: '', email: '', app_id: 'marketops', permission: 'read' });
  const allowed = !authEnabled || hasPlatformAdmin(claims);
  const load = async () => {
    try {
      const response = await fetch(`/v1/administration/access?tenant_id=${encodeURIComponent(tenant)}`, { headers: headers() });
      if (!response.ok) throw Error(await response.text());
      setGrants((await response.json()).access ?? []);
    } catch (cause) { setError(String(cause)); }
  };
  useEffect(() => { if (allowed) void load(); }, [tenant, allowed]);
  if (!allowed) return <div className="rounded border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">Super-admin access is required.</div>;
  const search = async (event: React.FormEvent) => {
    event.preventDefault(); setError('');
    if (query.trim().length < 2) { setError('Enter at least two characters to search the identity directory.'); return; }
    setSearching(true);
    try {
      const response = await fetch(`/v1/administration/idp-users?query=${encodeURIComponent(query.trim())}`, { headers: headers() });
      if (!response.ok) throw Error(await response.text());
      setResults((await response.json()).users ?? []);
    } catch (cause) { setError(String(cause)); setResults([]); } finally { setSearching(false); }
  };
  const choose = (user: DirectoryUser) => { setForm({ ...form, subject: user.subject, display_name: user.display_name || user.username, email: user.email }); setResults([]); };
  const save = async (event: React.FormEvent) => {
    event.preventDefault(); setError('');
    const response = await fetch('/v1/administration/access', { method: 'PUT', headers: headers(), body: JSON.stringify({ ...form, tenant_id: tenant }) });
    if (!response.ok) { setError(await response.text()); return; }
    setForm({ subject: '', display_name: '', email: '', app_id: 'marketops', permission: 'read' }); void load();
  };
  const revoke = async (grant: Grant) => {
    const response = await fetch(`/v1/administration/access/${encodeURIComponent(grant.subject)}/${grant.app_id}?tenant_id=${encodeURIComponent(tenant)}`, { method: 'DELETE', headers: headers() });
    if (!response.ok) { setError(await response.text()); return; } void load();
  };
  return <div className="space-y-3"><div><h1 className="text-lg font-semibold">Use-case Access</h1><p className="text-xs text-gray-500">Find an existing Keycloak identity, then assign tenant-scoped MarketOps or CyberOps access.</p></div>
    <section className="rounded border border-gray-200 bg-white p-3"><div className="mb-2"><h2 className="text-sm font-semibold">Identity directory</h2><p className="text-xs text-gray-500">Search Keycloak by name, username, or email. Select an active identity to populate the access grant.</p></div><form onSubmit={search} className="flex flex-wrap items-end gap-2"><label className="text-xs font-medium text-gray-700">Search user<input value={query} onChange={event => setQuery(event.target.value)} placeholder="Name, username, or email" className="mt-1 block w-64 rounded border border-gray-300 px-2 py-1 text-sm" /></label><button disabled={searching} className="rounded border border-brand-600 px-3 py-1.5 text-sm font-medium text-brand-700 hover:bg-brand-50 disabled:opacity-50">{searching ? 'Searching…' : 'Search IdP'}</button></form>{results.length ? <div className="mt-3 divide-y rounded border border-gray-200">{results.map(user => <button key={user.subject} type="button" disabled={!user.enabled} onClick={() => choose(user)} className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm hover:bg-brand-50 disabled:cursor-not-allowed disabled:opacity-50"><span><strong>{user.display_name || user.username}</strong><span className="ml-2 text-xs text-gray-500">{user.username}{user.email ? ` · ${user.email}` : ''}</span></span><span className={user.enabled ? 'text-xs text-green-700' : 'text-xs text-gray-500'}>{user.enabled ? 'Select' : 'Disabled'}</span></button>)}</div> : null}</section>
    <form onSubmit={save} className="flex flex-wrap items-end gap-2 rounded border border-gray-200 bg-white p-3"><label className="text-xs">Selected identity<input required readOnly value={form.subject} placeholder="Select from IdP search" className="mt-1 block w-64 rounded border border-gray-300 bg-gray-50 p-1 font-mono text-xs" /></label><label className="text-xs">Display name<input readOnly value={form.display_name} className="mt-1 block rounded border border-gray-300 bg-gray-50 p-1" /></label><label className="text-xs">Email<input readOnly value={form.email} className="mt-1 block rounded border border-gray-300 bg-gray-50 p-1" /></label><select value={form.app_id} onChange={event => setForm({ ...form, app_id: event.target.value })} className="rounded border p-1 text-sm">{grantableProfiles.map((profile) => <option key={profile.app_id} value={profile.app_id}>{profile.label}</option>)}</select><select value={form.permission} onChange={event => setForm({ ...form, permission: event.target.value })} className="rounded border p-1 text-sm"><option value="read">Read</option><option value="write">Write</option></select><button disabled={!form.subject} className="rounded bg-brand-700 px-3 py-1.5 text-sm text-white disabled:opacity-50">Save access</button></form>
    {error && <p className="rounded border border-red-200 bg-red-50 p-2 text-sm text-red-700">{error}</p>}
    <div className="overflow-x-auto rounded border border-gray-200 bg-white"><table className="min-w-full text-sm"><thead className="bg-gray-50 text-left text-xs text-gray-500"><tr><th className="p-2">Identity</th><th className="p-2">Use case</th><th className="p-2">Access</th><th className="p-2"></th></tr></thead><tbody>{grants.map(grant => <tr key={`${grant.subject}-${grant.app_id}`} className="border-t"><td className="p-2"><div>{grant.display_name || grant.subject}</div><code className="text-xs text-gray-500">{grant.subject}</code></td><td className="p-2">{grant.app_id}</td><td className="p-2">{grant.permission}</td><td className="p-2"><button type="button" onClick={() => void revoke(grant)} className="text-xs text-red-700">Revoke</button></td></tr>)}</tbody></table></div>
  </div>;
}
