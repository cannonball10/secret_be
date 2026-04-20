// REPLICANT — HOST TV SCREENS
// 1920x1080 canvases shown on the shared TV/desktop display.

// ─────────────────────────────────────────────────────────
// 1. LOBBY — waiting for players to join
// ─────────────────────────────────────────────────────────
function HostLobby() {
  const players = [
    { n:'MARA',    num:'01', portrait:'M', ready:true,  accent:rpColors.ink },
    { n:'DEV',     num:'02', portrait:'D', ready:true,  accent:rpColors.stampBlue },
    { n:'PRIYA',   num:'03', portrait:'P', ready:true,  accent:rpColors.ink },
    { n:'JORDAN',  num:'04', portrait:'J', ready:false, accent:rpColors.inkFaded },
    { n:'LEO',     num:'05', portrait:'L', ready:true,  accent:rpColors.ink },
    { n:'SAM',     num:'06', portrait:'S', ready:true,  accent:rpColors.stampRed },
    { n:'NINA',    num:'07', portrait:'N', ready:false, accent:rpColors.inkFaded },
    { n:'RIO',     num:'08', portrait:'R', ready:true,  accent:rpColors.ink },
  ];
  return (
    <RPTVChrome title="CANDIDATE INTAKE" nodeId="NODE 04-7" phase="PRE-DEPLOY">
      <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', height:'100%', gap:0 }}>
        {/* LEFT — join instructions */}
        <div style={{ padding:'56px 72px', display:'flex', flexDirection:'column', gap:36, borderRight:`1px solid ${rpColors.broadcastRule}` }}>
          <div>
            <div className="t-eyebrow" style={{ color:rpColors.cyan, marginBottom:14 }}>◼ DEPT. OF HUMAN AFFAIRS · FORM R-07</div>
            <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:120, lineHeight:0.9, letterSpacing:'-0.02em', position:'relative' }}>
              <span style={{ position:'absolute', left:6, top:6, color:rpColors.stampRed, opacity:0.5 }}>REPLICANT</span>
              <span style={{ color:rpColors.paper3, position:'relative' }}>REPLICANT</span>
            </div>
            <div className="t-memo" style={{ color:rpColors.paper3, marginTop:16, fontSize:18, opacity:0.8, maxWidth:520 }}>
              A Human Verification Procedure, in Eight Rounds.
            </div>
          </div>

          <div style={{ border:`1px solid ${rpColors.broadcastRule}`, padding:'24px 28px', background:'rgba(95,211,204,0.04)' }}>
            <div className="t-eyebrow" style={{ color:rpColors.cyan, marginBottom:12 }}>▸ INTAKE INSTRUCTIONS</div>
            <ol style={{ margin:0, padding:'0 0 0 22px', fontFamily:'Special Elite, monospace', fontSize:17, lineHeight:1.9, color:rpColors.paper3 }}>
              <li>Open <span style={{ color:rpColors.cyan }}>replicant.game</span> on your device.</li>
              <li>Submit the six-character session code.</li>
              <li>Await role allocation. Do not discuss assignment.</li>
            </ol>
          </div>

          <div style={{ display:'flex', alignItems:'center', gap:24 }}>
            <div>
              <div className="t-eyebrow" style={{ color:rpColors.inkFaded, fontSize:11, marginBottom:6 }}>SESSION CODE</div>
              <div style={{ fontFamily:'JetBrains Mono, monospace', fontWeight:700, fontSize:64, letterSpacing:'0.15em', color:rpColors.cyan, lineHeight:1, textShadow:`0 0 20px ${rpColors.cyan}66` }}>R07-MKQ</div>
            </div>
            <div style={{ width:120, height:120, background:rpColors.paper3, padding:8 }}>
              {/* QR placeholder */}
              <div style={{ width:'100%', height:'100%',
                backgroundImage:`radial-gradient(${rpColors.ink} 2px, transparent 2px), radial-gradient(${rpColors.ink} 2px, transparent 2px)`,
                backgroundSize:'10px 10px, 10px 10px',
                backgroundPosition:'0 0, 5px 5px',
              }}/>
            </div>
          </div>
        </div>

        {/* RIGHT — player roster */}
        <div style={{ padding:'56px 72px', display:'flex', flexDirection:'column', gap:28 }}>
          <div style={{ display:'flex', justifyContent:'space-between', alignItems:'flex-end' }}>
            <div>
              <div className="t-eyebrow" style={{ color:rpColors.cyan, marginBottom:6 }}>CANDIDATES REGISTERED</div>
              <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:72, color:rpColors.paper3, lineHeight:0.9 }}>
                06<span style={{ color:rpColors.inkFaded, fontSize:40 }}>/08</span>
              </div>
            </div>
            <div style={{ fontFamily:'JetBrains Mono, monospace', fontSize:13, color:rpColors.inkFaded, textAlign:'right', lineHeight:1.6 }}>
              MIN. THRESHOLD: 05<br/>
              MAX. CAPACITY: 10<br/>
              <span style={{ color:rpColors.stampGreen }}>● QUORUM ACHIEVED</span>
            </div>
          </div>

          <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap:10 }}>
            {players.map(p => (
              <div key={p.num} style={{
                display:'flex', alignItems:'center', gap:12,
                border:`1px solid ${p.ready?rpColors.cyan:rpColors.broadcastRule}`,
                background: p.ready?'rgba(95,211,204,0.06)':'rgba(255,255,255,0.02)',
                padding:'10px 14px', opacity:p.ready?1:0.55,
              }}>
                <div style={{
                  width:44, height:44, flexShrink:0,
                  background: p.ready?rpColors.cyan:rpColors.broadcastRule, color:rpColors.broadcast,
                  display:'flex', alignItems:'center', justifyContent:'center',
                  fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:22,
                }}>{p.portrait}</div>
                <div style={{ flex:1 }}>
                  <div style={{ fontFamily:'JetBrains Mono, monospace', fontSize:10, color:rpColors.inkFaded, letterSpacing:1.4 }}>SUBJECT #{p.num}</div>
                  <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:600, fontSize:18, color:rpColors.paper3, letterSpacing:'0.02em' }}>{p.n}</div>
                </div>
                <div style={{ fontFamily:'JetBrains Mono, monospace', fontSize:10, color:p.ready?rpColors.cyan:rpColors.inkFaded }}>
                  {p.ready ? '● READY' : '○ JOINING…'}
                </div>
              </div>
            ))}
          </div>

          <div style={{ marginTop:'auto', borderTop:`1px solid ${rpColors.broadcastRule}`, paddingTop:20, display:'flex', justifyContent:'space-between', alignItems:'center' }}>
            <div style={{ fontFamily:'Special Elite, monospace', fontSize:15, color:rpColors.paper3, opacity:0.7 }}>
              "Awaiting final two citizens. The Department does not tolerate tardiness."
            </div>
            <div style={{ background:rpColors.cyan, color:rpColors.broadcast, padding:'10px 22px', fontFamily:'Oswald, Impact, sans-serif', fontSize:16, letterSpacing:'0.1em', fontWeight:700 }}>
              DEPLOY ▸
            </div>
          </div>
        </div>
      </div>
    </RPTVChrome>
  );
}

// ─────────────────────────────────────────────────────────
// 2. NARRATOR SPEAKING — the AI host monologue
// ─────────────────────────────────────────────────────────
function HostNarrator() {
  return (
    <RPTVChrome title="THE DEPARTMENT SPEAKS" nodeId="VOICE CH-01" phase="DAY 02 · 14:02">
      <div style={{ height:'100%', display:'flex', flexDirection:'column', justifyContent:'center', padding:'40px 120px', position:'relative' }}>
        <div className="t-eyebrow" style={{ color:rpColors.cyan, marginBottom:16 }}>▸ AUDIO TRANSMISSION · SYNTHESIZED VOICE · NODE 04-7</div>

        {/* waveform */}
        <div style={{ display:'flex', alignItems:'center', gap:5, height:60, marginBottom:48 }}>
          {Array.from({length:80}).map((_,i) => {
            const h = 6 + Math.abs(Math.sin(i*0.4)*Math.cos(i*0.17))*54;
            return <div key={i} style={{ width:6, height:h, background: i<60?rpColors.cyan:rpColors.broadcastRule, opacity: i<60?1:0.4 }}/>;
          })}
        </div>

        <div style={{ fontFamily:'Special Elite, monospace', fontSize:54, lineHeight:1.35, color:rpColors.paper3, letterSpacing:'-0.005em', maxWidth:1400 }}>
          "Good afternoon, citizens. Yesterday's termination of <span style={{ color:rpColors.stampRed, textDecoration:`underline ${rpColors.stampRed} 2px`, textUnderlineOffset:8 }}>Subject MARA</span> has been processed. Records indicate she was, regrettably,<br/><span style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, color:rpColors.cyan, fontSize:72, letterSpacing:'0.02em' }}>HUMAN.</span>"
        </div>

        <div style={{ position:'absolute', bottom:40, left:120, right:120, display:'flex', justifyContent:'space-between', alignItems:'flex-end' }}>
          <div style={{ fontFamily:'JetBrains Mono, monospace', fontSize:13, color:rpColors.inkFaded, lineHeight:1.6 }}>
            <div style={{ color:rpColors.cyan }}>▸ SPEAKING</div>
            <div>BUFFER: 74%</div>
            <div>SYNTH. MODEL: ORACLE-III</div>
          </div>
          <div style={{ display:'flex', gap:12 }}>
            <div style={{ border:`1px solid ${rpColors.broadcastRule}`, padding:'8px 14px', fontFamily:'JetBrains Mono, monospace', fontSize:11, color:rpColors.inkFaded }}>⏸ INTERRUPT</div>
            <div style={{ border:`1px solid ${rpColors.broadcastRule}`, padding:'8px 14px', fontFamily:'JetBrains Mono, monospace', fontSize:11, color:rpColors.inkFaded }}>↺ REPEAT</div>
            <div style={{ border:`1px solid ${rpColors.cyan}`, padding:'8px 14px', fontFamily:'JetBrains Mono, monospace', fontSize:11, color:rpColors.cyan }}>▸ CONTINUE</div>
          </div>
        </div>
      </div>
    </RPTVChrome>
  );
}

// ─────────────────────────────────────────────────────────
// 3. NIGHT PHASE — task assignment
// ─────────────────────────────────────────────────────────
function HostNight() {
  return (
    <RPTVChrome title="NIGHT CYCLE · LIGHTS OUT" nodeId="NODE 04-7" phase="NIGHT 02">
      <div style={{ height:'100%', display:'flex', flexDirection:'column', alignItems:'center', justifyContent:'center', gap:40, padding:60 }}>
        <div className="t-eyebrow" style={{ color:rpColors.stampRed, letterSpacing:'0.3em' }}>▼ LIGHTS OUT ▼</div>

        <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:240, letterSpacing:'-0.02em', lineHeight:0.9, color:rpColors.paper3, position:'relative' }}>
          <span style={{ position:'absolute', left:12, top:12, color:rpColors.stampRed, opacity:0.5 }}>NIGHT</span>
          <span style={{ position:'relative' }}>NIGHT</span>
        </div>

        <div style={{ display:'grid', gridTemplateColumns:'repeat(3, 320px)', gap:24, marginTop:20 }}>
          {[
            { eyb:'HUMANS', txt:'Complete your assigned task.\nTouch nothing else.', c:rpColors.cyan },
            { eyb:'REPLICANTS', txt:'Select one citizen for\ndeactivation.', c:rpColors.stampRed },
            { eyb:'SINGULARITY', txt:'Observe. Intervene once\nper cycle.', c:rpColors.amber },
          ].map((card,i) => (
            <div key={i} style={{ border:`1px solid ${card.c}`, padding:24, background:'rgba(255,255,255,0.02)' }}>
              <div className="t-eyebrow" style={{ color:card.c, marginBottom:12 }}>◼ {card.eyb}</div>
              <div style={{ fontFamily:'Special Elite, monospace', fontSize:18, color:rpColors.paper3, lineHeight:1.5, whiteSpace:'pre-line' }}>{card.txt}</div>
            </div>
          ))}
        </div>

        <div style={{ marginTop:30, display:'flex', alignItems:'center', gap:40 }}>
          <RPTimer value="01:30" label="CYCLE ENDS IN" />
          <div style={{ fontFamily:'Special Elite, monospace', fontSize:17, color:rpColors.paper3, opacity:0.7, maxWidth:420 }}>
            "Close your eyes. Or don't. The Department sees everything regardless."
          </div>
        </div>
      </div>
    </RPTVChrome>
  );
}

// ─────────────────────────────────────────────────────────
// 4. VOTING RESULTS REVEAL
// ─────────────────────────────────────────────────────────
function HostResults() {
  const votes = [
    { n:'SAM',     num:'06', count:5, portrait:'S', elim:true,  accent:rpColors.stampRed },
    { n:'PRIYA',   num:'03', count:2, portrait:'P', elim:false, accent:rpColors.ink },
    { n:'JORDAN',  num:'04', count:1, portrait:'J', elim:false, accent:rpColors.ink },
    { n:'LEO',     num:'05', count:0, portrait:'L', elim:false, accent:rpColors.ink },
  ];
  const max = Math.max(...votes.map(v=>v.count));
  return (
    <RPTVChrome title="TRIBUNAL RESULT · VERDICT ISSUED" nodeId="NODE 04-7" phase="DAY 02 · 16:44">
      <div style={{ height:'100%', padding:'48px 80px', display:'grid', gridTemplateColumns:'1.2fr 1fr', gap:60 }}>
        {/* LEFT — verdict */}
        <div style={{ display:'flex', flexDirection:'column', justifyContent:'center', position:'relative' }}>
          <div className="t-eyebrow" style={{ color:rpColors.cyan, marginBottom:10 }}>▸ VERDICT OF THE ASSEMBLED CITIZENS</div>
          <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:80, color:rpColors.paper3, lineHeight:0.9, letterSpacing:'-0.02em' }}>
            SUBJECT #06
          </div>
          <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:180, color:rpColors.stampRed, lineHeight:0.9, marginTop:8, letterSpacing:'-0.03em' }}>
            SAM
          </div>
          <div style={{ fontFamily:'Special Elite, monospace', fontSize:22, color:rpColors.paper3, opacity:0.75, marginTop:16 }}>
            …has been selected for TERMINATION.
          </div>

          <div style={{ position:'absolute', bottom:0, right:0, transform:'rotate(-8deg)' }}>
            <div style={{
              border:`4px solid ${rpColors.stampRed}`, color:rpColors.stampRed,
              fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:48, letterSpacing:'0.08em',
              padding:'10px 26px', textTransform:'uppercase',
              mixBlendMode:'screen', opacity:0.9,
            }}>TERMINATED</div>
          </div>

          <div style={{ marginTop:40, display:'flex', gap:20, alignItems:'center' }}>
            <div>
              <div className="t-eyebrow" style={{ color:rpColors.inkFaded, marginBottom:4 }}>TRUE IDENTITY</div>
              <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontSize:44, fontWeight:700, color:rpColors.cyan, letterSpacing:'0.02em' }}>REPLICANT</div>
            </div>
            <div style={{ height:50, width:1, background:rpColors.broadcastRule }}/>
            <div style={{ fontFamily:'Special Elite, monospace', fontSize:15, color:rpColors.paper3, opacity:0.7, maxWidth:280 }}>
              "A satisfactory outcome. The Department notes your intuition."
            </div>
          </div>
        </div>

        {/* RIGHT — tally */}
        <div style={{ display:'flex', flexDirection:'column', gap:20, justifyContent:'center' }}>
          <div className="t-eyebrow" style={{ color:rpColors.cyan }}>TALLY OF BALLOTS · 08 CAST</div>
          <div style={{ display:'flex', flexDirection:'column', gap:14 }}>
            {votes.map(v => (
              <div key={v.num} style={{ display:'flex', alignItems:'center', gap:14 }}>
                <div style={{ width:56, height:56, flexShrink:0,
                  background: v.elim?rpColors.stampRed:rpColors.broadcastRule,
                  color: v.elim?rpColors.paper3:rpColors.paper3,
                  display:'flex', alignItems:'center', justifyContent:'center',
                  fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:26,
                  border: v.elim?`2px solid ${rpColors.stampRed}`:`1px solid ${rpColors.broadcastRule}`,
                }}>{v.portrait}</div>
                <div style={{ flex:1 }}>
                  <div style={{ display:'flex', justifyContent:'space-between', alignItems:'baseline' }}>
                    <span style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:600, fontSize:20, color:rpColors.paper3, letterSpacing:'0.02em' }}>{v.n}</span>
                    <span style={{ fontFamily:'JetBrains Mono, monospace', fontSize:20, fontWeight:700, color: v.elim?rpColors.stampRed:rpColors.paper3 }}>{String(v.count).padStart(2,'0')}</span>
                  </div>
                  <div style={{ height:10, background:rpColors.broadcast2, marginTop:4, position:'relative', border:`1px solid ${rpColors.broadcastRule}` }}>
                    <div style={{ height:'100%', width:`${(v.count/max)*100}%`,
                      background: v.elim? `repeating-linear-gradient(45deg, ${rpColors.stampRed} 0 6px, ${rpColors.stampRed2} 6px 12px)` : rpColors.cyan,
                    }}/>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div style={{ marginTop:16, border:`1px solid ${rpColors.broadcastRule}`, padding:'14px 18px', background:'rgba(255,255,255,0.02)' }}>
            <div className="t-eyebrow" style={{ color:rpColors.inkFaded, marginBottom:6 }}>POPULATION STATUS</div>
            <div style={{ display:'flex', gap:24, fontFamily:'JetBrains Mono, monospace', fontSize:14, color:rpColors.paper3 }}>
              <span>HUMANS <b style={{ color:rpColors.cyan }}>04</b></span>
              <span>REPLICANTS <b style={{ color:rpColors.stampRed }}>02</b></span>
              <span>SINGULARITY <b style={{ color:rpColors.amber }}>01</b></span>
              <span style={{ color:rpColors.inkFaded }}>TERMINATED <b>01</b></span>
            </div>
          </div>
        </div>
      </div>
    </RPTVChrome>
  );
}

Object.assign(window, { HostLobby, HostNarrator, HostNight, HostResults });
