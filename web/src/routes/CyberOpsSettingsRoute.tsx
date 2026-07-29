import { useEffect, useState } from "react";
import { useTenant } from "../auth/session";
import { useCyberOpsIoTNetworkConfig, useUpdateCyberOpsIoTNetworkConfig } from "../api/queries";

export function CyberOpsSettingsRoute(){
 const tenant=useTenant(); const config=useCyberOpsIoTNetworkConfig(tenant); const apply=useUpdateCyberOpsIoTNetworkConfig(tenant); const [cidrs,setCidrs]=useState("");
 useEffect(()=>{if(config.data) setCidrs(config.data.network_config.internal_cidrs.join("\n"));},[config.data]);
 const submit=()=>apply.mutate(cidrs.split(/[\n,]/).map(value=>value.trim()).filter(Boolean));
 return <div className="max-w-3xl space-y-4"><div><h1 className="text-lg font-semibold">CyberOps settings</h1><p className="text-xs text-gray-500">Enter one internal network CIDR per line, then apply the configuration.</p></div><section className="rounded border border-gray-200 bg-white p-4 space-y-3"><label className="block text-sm font-medium" htmlFor="iot-cidrs">Internal network CIDRs</label><textarea id="iot-cidrs" value={cidrs} onChange={event=>setCidrs(event.target.value)} rows={7} className="w-full rounded border border-gray-300 p-2 font-mono text-sm" placeholder={"192.168.1.0/24\n10.0.0.0/16"}/><button type="button" onClick={submit} disabled={apply.isPending} className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50">{apply.isPending?"Applying CIDRs...":"Apply CIDRs"}</button>{config.isError?<p className="text-sm text-amber-700">Current configuration could not be loaded; you can still apply the values above.</p>:null}{apply.isSuccess?<p className="text-sm text-emerald-700">CIDR configuration applied.</p>:null}{apply.isError?<p className="text-sm text-red-700">Unable to apply CIDRs: {String(apply.error)}</p>:null}</section></div>;
}
