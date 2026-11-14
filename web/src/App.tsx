import { useMemo, useState } from 'react';
import React from 'react';
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

type Result = {
  target: string; started_at: string; elapsed_seconds: number; scheduled_requests: number;
  completed_requests: number; successful_requests: number; failed_requests: number;
  network_errors: number; http_error_responses: number; requests_per_second: number;
  bytes_read: number;
  latency: { average_ms: number; min_ms: number; max_ms: number; p50_ms: number; p90_ms: number; p95_ms: number; p99_ms: number };
  status_codes: Record<string, number>; backend_signals: Record<string, Record<string, number>>; notes: string[];
};
type ApiError = { error?: string };
const number = (v:number) => new Intl.NumberFormat().format(v);
const ms = (v:number) => v.toFixed(1) + ' ms';

export default function App() {
  const [url,setUrl]=useState('https://');
  const [concurrency,setConcurrency]=useState(5);
  const [duration,setDuration]=useState(10);
  const [rps,setRps]=useState(10);
  const [maxRequests,setMaxRequests]=useState(1000);
  const [authorized,setAuthorized]=useState(false);
  const [busy,setBusy]=useState(false);
  const [error,setError]=useState('');
  const [result,setResult]=useState<Result|null>(null);
  const chartData=useMemo(()=>result?[
    {metric:'Average',ms:result.latency.average_ms},{metric:'p50',ms:result.latency.p50_ms},
    {metric:'p90',ms:result.latency.p90_ms},{metric:'p95',ms:result.latency.p95_ms},
    {metric:'p99',ms:result.latency.p99_ms},{metric:'Max',ms:result.latency.max_ms},
  ]:[],[result]);

  async function startRun(event:React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError(''); setResult(null); setBusy(true);
    try {
      const response=await fetch('/api/v1/runs',{method:'POST',headers:{'Content-Type':'application/json'},
        body:JSON.stringify({url:url.trim(),concurrency,duration_seconds:duration,requests_per_second:rps,max_requests:maxRequests,authorization_confirmed:authorized})});
      const data=await response.json() as Result|ApiError;
      if(!response.ok) throw new Error((data as ApiError).error??'Request failed ('+response.status+')');
      setResult(data as Result);
    } catch(err) {setError(err instanceof Error?err.message:'The test could not be completed.');}
    finally {setBusy(false);}
  }

  function exportReport() {
    if(!result)return;
    const blob=new Blob([JSON.stringify(result,null,2)],{type:'application/json'});
    const link=document.createElement('a');link.href=URL.createObjectURL(blob);link.download='loadforge-report.json';link.click();URL.revokeObjectURL(link.href);
  }

  return <main className="shell">
    <header className="topbar"><a className="brand" href="#top"><span className="brand-mark">LF</span>LoadForge</a><span className="version">DEVELOPER PERFORMANCE TOOLKIT · v0.1.0</span></header>
    <section className="hero" id="top"><div className="eyebrow">MEASURE THE SYSTEM. NOT THE ASSUMPTION.</div><h1>Find the limit before<br/><em>your users do.</em></h1><p>Run bounded HTTP load tests, inspect latency under pressure, and look for observable backend distribution signals.</p></section>
    <section className="workspace">
      <form className="panel config-panel" onSubmit={startRun}>
        <div className="panel-head"><div><div className="section-kicker">01 / CONFIGURE</div><h2>Test configuration</h2></div><span className="limit-tag">SAFE LIMITS ON</span></div>
        <label className="field-label" htmlFor="target">Target URL</label>
        <input id="target" className="target-input" type="url" required value={url} onChange={e=>setUrl(e.target.value)} placeholder="https://api.yourproduct.com/health"/>
        <div className="field-grid">
          <label><span className="field-label">Concurrency <b>{concurrency}</b></span><input type="range" min="1" max="25" value={concurrency} onChange={e=>setConcurrency(Number(e.target.value))}/><small>Virtual workers · max 25</small></label>
          <label><span className="field-label">Duration <b>{duration}s</b></span><input type="range" min="1" max="30" value={duration} onChange={e=>setDuration(Number(e.target.value))}/><small>Run window · max 30s</small></label>
          <label><span className="field-label">Request rate <b>{rps} / sec</b></span><input type="range" min="1" max="50" value={rps} onChange={e=>setRps(Number(e.target.value))}/><small>Scheduled rate · max 50 RPS</small></label>
          <label><span className="field-label">Request cap <b>{number(maxRequests)}</b></span><input type="range" min="100" max="5000" step="100" value={maxRequests} onChange={e=>setMaxRequests(Number(e.target.value))}/><small>Hard ceiling · max 5,000</small></label>
        </div>
        <label className="consent"><input type="checkbox" checked={authorized} onChange={e=>setAuthorized(e.target.checked)}/><span>I own this target or have explicit permission to run a performance test against it.</span></label>
        <button className="run-button" type="submit" disabled={busy||!authorized}>{busy?'Running bounded test…':'Run performance test ↗'}</button>
        <div className="form-foot">GET requests only · Redirects not followed · Per-request timeout 5s · No automatic retries</div>
      </form>
      <aside className="side-panel"><div className="section-kicker">WHAT YOU WILL MEASURE</div>
        <div className="measure-item"><span className="measure-number">01</span><div><b>Latency distribution</b><p>Average, p50, p90, p95, p99 and maximum response time.</p></div></div>
        <div className="measure-item"><span className="measure-number">02</span><div><b>Throughput and errors</b><p>Completed requests per second, status codes, timeouts and failures.</p></div></div>
        <div className="measure-item"><span className="measure-number">03</span><div><b>Backend signals</b><p>Counts of instance-related response headers when exposed.</p></div></div>
        <div className="notice"><b>Important</b><p>A URL cannot reveal a backend's true maximum capacity. Results describe this test run, from this machine, over this network.</p></div>
      </aside>
    </section>
    {error&&<div className="error-banner" role="alert">{error}</div>}
    {busy&&<div className="running-state">Load generation is active. The report will appear when the run completes.</div>}
    {result&&<section className="results" aria-live="polite">
      <div className="results-heading"><div><div className="section-kicker">02 / RESULTS</div><h2>Run report</h2><p className="result-target">{result.target}</p></div><span className="complete-tag">COMPLETE</span></div>
      <div className="metric-grid">
        <Metric label="Requests / second" value={result.requests_per_second.toFixed(2)} detail={number(result.completed_requests)+' completed'}/>
        <Metric label="p95 latency" value={ms(result.latency.p95_ms)} detail={'p99 '+ms(result.latency.p99_ms)}/>
        <Metric label="Successful requests" value={number(result.successful_requests)} detail={result.failed_requests+' failed'}/>
        <Metric label="Test duration" value={result.elapsed_seconds.toFixed(2)+'s'} detail={number(result.scheduled_requests)+' scheduled'}/>
      </div>
      <div className="result-columns">
        <div className="panel result-panel"><div className="panel-head"><div><div className="section-kicker">LATENCY PROFILE</div><h3>Response time percentiles</h3></div></div><div className="chart-wrap"><ResponsiveContainer width="100%" height="100%"><LineChart data={chartData} margin={{top:10,right:14,left:0,bottom:4}}><CartesianGrid strokeDasharray="3 4" stroke="#263342" vertical={false}/><XAxis dataKey="metric" tick={{fill:'#8e9aaa',fontSize:11}} axisLine={false} tickLine={false}/><YAxis tick={{fill:'#8e9aaa',fontSize:11}} axisLine={false} tickLine={false} width={48}/><Tooltip contentStyle={{background:'#111b29',border:'1px solid #314052',borderRadius:8}} formatter={(value)=>[Number(value).toFixed(2)+' ms','Latency']}/><Line type="monotone" dataKey="ms" stroke="#56d6b1" strokeWidth={2.5} dot={{r:4,fill:'#56d6b1',stroke:'#101925',strokeWidth:2}}/></LineChart></ResponsiveContainer></div></div>
        <div className="panel result-panel"><div className="panel-head"><div><div className="section-kicker">HTTP RESPONSES</div><h3>Status code counts</h3></div></div><div className="status-list">{Object.entries(result.status_codes).sort(([a],[b])=>Number(a)-Number(b)).map(([status,count])=><div className="status-row" key={status}><span className={Number(status)>=400?'status-code bad':'status-code'}>{status}</span><div className="status-track"><span style={{width:Math.max(2,count/Math.max(1,result.completed_requests)*100)+'%'}}/></div><b>{number(count)}</b></div>)}</div><div className="small-stats"><span>Network errors <b>{result.network_errors}</b></span><span>HTTP error responses <b>{result.http_error_responses}</b></span><span>Bytes sampled <b>{number(result.bytes_read)}</b></span></div></div>
      </div>
      <div className="panel signal-panel"><div className="panel-head"><div><div className="section-kicker">INFRASTRUCTURE OBSERVATIONS</div><h3>Backend response signals</h3></div></div>
        {Object.keys(result.backend_signals).length?<div className="signal-grid">{Object.entries(result.backend_signals).map(([header,values])=><div className="signal-card" key={header}><b>{header}</b>{Object.entries(values).map(([value,count])=><div className="signal-value" key={value}><code>{value}</code><span>{count} responses</span></div>)}</div>)}</div>:<p className="muted">No recognized backend-identifying headers were observed. A load balancer may still exist; the target may simply not expose instance identity.</p>}
      </div>
      {result.notes.map((note,index)=><p className="note-line" key={index}>↳ {note}</p>)}
      <div className="result-actions"><button className="secondary-button" onClick={exportReport}>Export JSON report ↓</button><button className="secondary-button" onClick={()=>{setResult(null);setError('');window.scrollTo({top:0,behavior:'smooth'});}}>New test ↗</button></div>
    </section>}
    <footer><span>LOADFORGE · OPEN SOURCE · GO ENGINE</span><span>Only test systems you own or are authorized to assess.</span></footer>
  </main>;
}
function Metric({label,value,detail}:{label:string;value:string;detail:string}) {
  return <article className="metric-card"><span>{label}</span><strong>{value}</strong><small>{detail}</small></article>;
}
