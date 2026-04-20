// REPLICANT — MOBILE SCREENS (the "personal dossier" form factor)

// Shared screen chrome: paper-textured content area
function MScreen({ children, bg='paper' }) {
  return (
    <div className="paper-tex" style={{
      width:'100%', height:'100%', background: bg==='bright'?rpColors.paper3: bg==='pink'?rpColors.paper2:rpColors.paper,
      position:'relative', overflow:'hidden',
      display:'flex', flexDirection:'column',
    }}>
      {children}
    </div>
  );
}

// Header bar used on most mobile screens
function MHeader({ left='SUBJECT #04', right='DAY 02' }) {
  return (
    <div style={{ padding:'6px 20px 12px', display:'flex', justifyContent:'space-between', alignItems:'center', borderBottom:`1px dashed ${rpColors.paperLine}`, fontFamily:'JetBrains Mono, monospace', fontSize:10, color:rpColors.inkFaded, letterSpacing:1.5 }}>
      <span>■ {left}</span>
      <span>R-07 · {right}</span>
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// 1. ROLE REVEAL — dossier opened
// ─────────────────────────────────────────────────────────
function MobileRoleReveal() {
  return (
    <MScreen>
      <RPMobileStatusBar />
      <MHeader left="SUBJECT #04 · JORDAN" right="DAY 00 · INTAKE" />

      <div style={{ padding:'24px 22px 10px' }}>
        <div style={{ display:'flex', justifyContent:'space-between', alignItems:'flex-start' }}>
          <div>
            <div className="t-eyebrow" style={{ color:rpColors.inkFaded, fontSize:10 }}>CONFIDENTIAL · DOSSIER R-07/04</div>
            <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:42, color:rpColors.ink, lineHeight:0.95, marginTop:4 }}>ROLE<br/>ASSIGNMENT</div>
          </div>
          <RPSeal size={82} />
        </div>
      </div>

      {/* Role card */}
      <div style={{ margin:'16px 22px', padding:'22px 20px', background:rpColors.paper3, border:`2px solid ${rpColors.ink}`, boxShadow:'4px 4px 0 rgba(28,26,21,0.35)', position:'relative' }}>
        <div className="t-eyebrow" style={{ color:rpColors.inkFaded, fontSize:10, marginBottom:4 }}>CLASSIFICATION</div>
        <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:64, color:rpColors.stampRed, lineHeight:0.9, letterSpacing:'-0.01em', position:'relative' }}>
          <span style={{ position:'absolute', left:3, top:3, color:rpColors.ink, opacity:0.25 }}>REPLICANT</span>
          <span style={{ position:'relative' }}>REPLICANT</span>
        </div>
        <div style={{ marginTop:14, fontFamily:'Special Elite, monospace', fontSize:14, color:rpColors.ink, lineHeight:1.55 }}>
          You are a synthetic citizen, indistinguishable from the rest. Your directive: blend in. <span className="redacted">redirect suspicion</span>. Coordinate silently with your kin during Night Cycles to deactivate humans.
        </div>

        {/* stamp overlay */}
        <div style={{ position:'absolute', right:-6, top:-14 }}>
          <RPStamp variant="red" rotate={8} size={16}>CLASSIFIED</RPStamp>
        </div>
      </div>

      {/* Kin */}
      <div style={{ margin:'0 22px 10px', padding:'14px 16px', background:rpColors.paper2, border:`1px solid ${rpColors.ink}`, borderLeft:`6px solid ${rpColors.stampRed}` }}>
        <div className="t-eyebrow" style={{ color:rpColors.inkFaded, fontSize:10, marginBottom:8 }}>▸ KNOWN KIN · 01</div>
        <RPPlayerChip name="SAM" num="#06" status="ACTIVE" portrait="S" accent={rpColors.stampRed} small />
      </div>

      {/* Objective */}
      <div style={{ margin:'6px 22px 10px', fontFamily:'Special Elite, monospace', fontSize:13, color:rpColors.inkSoft, lineHeight:1.5 }}>
        <span className="t-eyebrow" style={{ color:rpColors.inkFaded, fontSize:10, letterSpacing:'0.22em', display:'block', marginBottom:6 }}>OBJECTIVE</span>
        Outnumber the humans. Be the last assembly standing when the count tips in your favor.
      </div>

      <div style={{ marginTop:'auto', padding:'16px 22px 22px', borderTop:`1px dashed ${rpColors.paperLine}` }}>
        <RPButton variant="stamp" full>I ACCEPT THE ASSIGNMENT ▸</RPButton>
        <div style={{ textAlign:'center', fontFamily:'JetBrains Mono, monospace', fontSize:10, color:rpColors.inkFaded, marginTop:10, letterSpacing:1.4 }}>
          HOLD TO REVEAL · RELEASE TO CONCEAL
        </div>
      </div>
    </MScreen>
  );
}

// ─────────────────────────────────────────────────────────
// 2. VOTING — tribunal ballot
// ─────────────────────────────────────────────────────────
function MobileVoting() {
  const candidates = [
    { n:'SAM',    num:'06', portrait:'S', accent:rpColors.stampRed, votes:3, selected:true },
    { n:'PRIYA',  num:'03', portrait:'P', accent:rpColors.ink,      votes:1 },
    { n:'JORDAN', num:'04', portrait:'J', accent:rpColors.stampBlue,votes:0, self:true },
    { n:'LEO',    num:'05', portrait:'L', accent:rpColors.ink,      votes:0 },
    { n:'NINA',   num:'07', portrait:'N', accent:rpColors.ink,      votes:1 },
    { n:'RIO',    num:'08', portrait:'R', accent:rpColors.ink,      votes:0 },
  ];
  return (
    <MScreen bg="bright">
      <RPMobileStatusBar />
      <MHeader left="SUBJECT #04 · JORDAN" right="DAY 02 · TRIBUNAL" />

      <div style={{ padding:'16px 22px 8px' }}>
        <div className="t-eyebrow" style={{ color:rpColors.stampRed, fontSize:10 }}>◼ OFFICIAL BALLOT · FORM V-02</div>
        <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:36, color:rpColors.ink, lineHeight:0.95, marginTop:4 }}>
          CAST YOUR<br/>VERDICT
        </div>
        <div style={{ fontFamily:'Special Elite, monospace', fontSize:13, color:rpColors.inkSoft, marginTop:8, lineHeight:1.5 }}>
          "Select one citizen for termination. The majority's choice will be processed immediately."
        </div>
      </div>

      <div style={{ margin:'10px 22px', display:'flex', alignItems:'center', justifyContent:'space-between', padding:'8px 12px', background:rpColors.paper2, border:`1px solid ${rpColors.ink}` }}>
        <RPTimer value="0:47" label="BALLOT CLOSES" danger />
        <div style={{ textAlign:'right', fontFamily:'JetBrains Mono, monospace', fontSize:10, color:rpColors.inkFaded, lineHeight:1.5 }}>
          CAST <b style={{ color:rpColors.ink, fontSize:14 }}>05</b>/08<br/>
          ABSTAIN <b style={{ color:rpColors.ink }}>01</b>
        </div>
      </div>

      <div style={{ flex:1, overflow:'auto', padding:'0 22px', display:'flex', flexDirection:'column', gap:8 }}>
        {candidates.map(c => (
          <div key={c.num} style={{
            display:'flex', alignItems:'center', gap:10,
            background: c.selected?rpColors.paper2:rpColors.paper3,
            border: `2px solid ${c.selected?rpColors.stampRed:rpColors.ink}`,
            borderLeft: c.selected?`10px solid ${rpColors.stampRed}`:`2px solid ${rpColors.ink}`,
            padding: c.selected?'10px 12px 10px 8px':'10px 12px',
            position:'relative', opacity: c.self?0.55:1,
          }}>
            <RPCheckbox checked={c.selected} />
            <div style={{
              width:42, height:42, background:c.accent, color:rpColors.paper3,
              border:`1.5px solid ${rpColors.ink}`,
              display:'flex', alignItems:'center', justifyContent:'center',
              fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:20,
            }}>{c.portrait}</div>
            <div style={{ flex:1 }}>
              <div style={{ fontFamily:'JetBrains Mono, monospace', fontSize:9, color:rpColors.inkFaded, letterSpacing:1.2 }}>SUBJECT #{c.num}</div>
              <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:600, fontSize:17, color:rpColors.ink, letterSpacing:'0.02em' }}>
                {c.n} {c.self && <span style={{ fontSize:10, color:rpColors.inkFaded, letterSpacing:1 }}>· YOU (INELIGIBLE)</span>}
              </div>
            </div>
            {c.votes > 0 && !c.self && (
              <div style={{ display:'flex', gap:2 }}>
                {Array.from({length:c.votes}).map((_,i)=>(
                  <div key={i} style={{ width:5, height:16, background:rpColors.stampRed }}/>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>

      <div style={{ padding:'14px 22px 22px', borderTop:`1px dashed ${rpColors.paperLine}`, background:rpColors.paper2 }}>
        <RPButton variant="stamp" full>
          <span style={{ fontFamily:'Special Elite, monospace', fontWeight:400, marginRight:6, letterSpacing:0 }}>◼</span>
          CAST BALLOT FOR SAM
        </RPButton>
        <div style={{ display:'flex', justifyContent:'space-between', fontFamily:'JetBrains Mono, monospace', fontSize:10, color:rpColors.inkFaded, marginTop:10, letterSpacing:1.2 }}>
          <span>[ ABSTAIN ]</span>
          <span>[ CHANGE SELECTION ]</span>
        </div>
      </div>
    </MScreen>
  );
}

// ─────────────────────────────────────────────────────────
// 3. CHAT / WHISPER — encrypted channel
// ─────────────────────────────────────────────────────────
function MobileChat() {
  const msgs = [
    { who:'SAM',    portrait:'S', accent:rpColors.stampRed, you:false, t:'21:44', body:'priya was in the server room when the lights cut. im telling you.' },
    { who:'YOU',    portrait:'J', accent:rpColors.stampBlue, you:true, t:'21:44', body:'thats a cover. she was with me.' },
    { who:'SAM',    portrait:'S', accent:rpColors.stampRed, you:false, t:'21:45', body:'redirect. the singularity watches. vote leo.' },
    { who:'YOU',    portrait:'J', accent:rpColors.stampBlue, you:true, t:'21:46', body:'too obvious. sleep on it.' },
    { who:'DEPT.',  portrait:'◼', accent:rpColors.ink,      you:false, system:true, t:'21:46', body:'CHANNEL CLOSES IN 00:45. TRANSCRIPT ARCHIVED.' },
  ];
  return (
    <MScreen>
      <RPMobileStatusBar />
      <MHeader left="WHISPER CHANNEL · KIN" right="ENC. AES-R7" />

      {/* banner */}
      <div style={{ background:rpColors.stampRed, color:rpColors.paper3, padding:'6px 20px', display:'flex', justifyContent:'space-between', alignItems:'center', fontFamily:'JetBrains Mono, monospace', fontSize:10, letterSpacing:1.4 }}>
        <span>▸ REPLICANT · PRIVATE</span>
        <span style={{ display:'flex', alignItems:'center', gap:6 }}><span className="animate-blink" style={{ width:6, height:6, background:rpColors.paper3, display:'inline-block' }}/> SWEEP IN 00:45</span>
      </div>

      <div style={{ flex:1, overflow:'auto', padding:'18px 20px', display:'flex', flexDirection:'column', gap:14 }}>
        {msgs.map((m,i) => m.system ? (
          <div key={i} style={{ textAlign:'center', fontFamily:'JetBrains Mono, monospace', fontSize:10, color:rpColors.inkFaded, letterSpacing:1.5, padding:'6px 0', borderTop:`1px dashed ${rpColors.paperLine}`, borderBottom:`1px dashed ${rpColors.paperLine}` }}>
            ◼ {m.body}
          </div>
        ) : (
          <div key={i} style={{ display:'flex', flexDirection: m.you?'row-reverse':'row', gap:8, alignItems:'flex-start' }}>
            <div style={{ width:30, height:30, background:m.accent, color:rpColors.paper3, border:`1.5px solid ${rpColors.ink}`, display:'flex', alignItems:'center', justifyContent:'center', fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:14, flexShrink:0 }}>{m.portrait}</div>
            <div style={{ maxWidth:'78%' }}>
              <div style={{ display:'flex', gap:8, alignItems:'baseline', marginBottom:3, flexDirection: m.you?'row-reverse':'row' }}>
                <span style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:600, fontSize:12, color:rpColors.ink, letterSpacing:'0.06em' }}>{m.who}</span>
                <span style={{ fontFamily:'JetBrains Mono, monospace', fontSize:9, color:rpColors.inkFaded }}>{m.t}</span>
              </div>
              <div style={{
                background: m.you?rpColors.ink:rpColors.paper3,
                color: m.you?rpColors.paper3:rpColors.ink,
                border: m.you?'none':`1.5px solid ${rpColors.ink}`,
                padding:'10px 13px',
                fontFamily:'Special Elite, monospace', fontSize:14, lineHeight:1.45,
                boxShadow: m.you?'none':'2px 2px 0 rgba(28,26,21,0.2)',
              }}>{m.body}</div>
            </div>
          </div>
        ))}

        {/* typing indicator */}
        <div style={{ display:'flex', gap:8, alignItems:'center', marginLeft:38 }}>
          <span className="t-eyebrow" style={{ fontSize:9, color:rpColors.inkFaded }}>SAM IS TYPING</span>
          <div style={{ display:'flex', gap:3 }}>
            {[0,0.2,0.4].map((d,i)=><span key={i} style={{ width:4, height:4, background:rpColors.stampRed, display:'inline-block', animation:`blink 1s ${d}s infinite` }}/>)}
          </div>
        </div>
      </div>

      <div style={{ padding:'10px 16px 18px', borderTop:`1px dashed ${rpColors.paperLine}`, background:rpColors.paper2 }}>
        <div style={{ display:'flex', gap:8, alignItems:'center', background:rpColors.paper3, border:`2px solid ${rpColors.ink}`, padding:'10px 12px' }}>
          <span style={{ fontFamily:'JetBrains Mono, monospace', fontSize:13, color:rpColors.inkFaded }}>▸</span>
          <span style={{ fontFamily:'Special Elite, monospace', fontSize:14, color:rpColors.ink, flex:1 }}>
            ok. i'll push jordan instead<span className="animate-blink" style={{ display:'inline-block', width:8, height:14, background:rpColors.ink, marginLeft:1, verticalAlign:'text-bottom' }}/>
          </span>
          <div style={{ background:rpColors.ink, color:rpColors.paper3, padding:'4px 10px', fontFamily:'Oswald, Impact, sans-serif', fontSize:12, fontWeight:700, letterSpacing:'0.1em' }}>SEND ◼</div>
        </div>
        <div style={{ textAlign:'center', fontFamily:'JetBrains Mono, monospace', fontSize:9, color:rpColors.inkFaded, marginTop:8, letterSpacing:1.3 }}>
          MESSAGES ARE ARCHIVED BY THE DEPARTMENT AFTER CYCLE END
        </div>
      </div>
    </MScreen>
  );
}

Object.assign(window, { MobileRoleReveal, MobileVoting, MobileChat });
