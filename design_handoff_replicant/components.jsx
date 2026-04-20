// REPLICANT — Shared components (global scope)
// Naming: every style block is prefixed `rp*` to avoid collision.

const rpColors = {
  paper:'#E8DFC9', paper2:'#D9CFB5', paper3:'#F2EAD3', paperLine:'#B8AE93',
  ink:'#1C1A15', inkSoft:'#3A352A', inkFaded:'#6B6450',
  broadcast:'#12120E', broadcast2:'#1E1D17', broadcastRule:'#3A3A2E',
  stampRed:'#B32724', stampRed2:'#8E1A18', stampGreen:'#3F6B3A', stampBlue:'#24426B',
  cyan:'#5FD3CC', cyanSoft:'#2A6E6A', amber:'#D69E2E',
};

// ─────────────────────────────────────────────────────────
// DEPARTMENT SEAL — the "brand mark"
// Circular stamp: outer ring "DEPT. OF HUMAN AFFAIRS", inner = R monogram
// ─────────────────────────────────────────────────────────
function RPSeal({ size = 120, color = rpColors.stampRed, rotate = -6 }) {
  const id = React.useId();
  return (
    <svg width={size} height={size} viewBox="0 0 120 120" style={{ transform: `rotate(${rotate}deg)`, mixBlendMode:'multiply', opacity:0.9 }}>
      <defs>
        <path id={id} d="M 60,60 m -44,0 a 44,44 0 1,1 88,0 a 44,44 0 1,1 -88,0" />
      </defs>
      <circle cx="60" cy="60" r="54" fill="none" stroke={color} strokeWidth="2" />
      <circle cx="60" cy="60" r="49" fill="none" stroke={color} strokeWidth="1" />
      <circle cx="60" cy="60" r="36" fill="none" stroke={color} strokeWidth="1.5" />
      <text fontFamily="Oswald, Impact, sans-serif" fontWeight="700" fontSize="9" letterSpacing="2" fill={color}>
        <textPath href={`#${id}`} startOffset="2%">DEPT. OF HUMAN AFFAIRS · FORM R-07 · CLASSIFIED ·</textPath>
      </text>
      <text x="60" y="58" textAnchor="middle" fontFamily="Oswald, Impact, sans-serif" fontWeight="700" fontSize="30" fill={color} letterSpacing="-1">R</text>
      <text x="60" y="74" textAnchor="middle" fontFamily="JetBrains Mono, monospace" fontSize="6" fill={color} letterSpacing="2">REPLICANT</text>
      <line x1="28" y1="60" x2="40" y2="60" stroke={color} strokeWidth="1" />
      <line x1="80" y1="60" x2="92" y2="60" stroke={color} strokeWidth="1" />
    </svg>
  );
}

// ─────────────────────────────────────────────────────────
// WORDMARK — "REPLICANT" with ghosted duplicate behind
// ─────────────────────────────────────────────────────────
function RPWordmark({ size = 72, color = rpColors.ink, ghost = rpColors.stampRed }) {
  return (
    <div style={{ position:'relative', display:'inline-block', fontFamily:'Oswald, Impact, sans-serif', fontWeight:700, fontSize:size, letterSpacing:'-0.01em', lineHeight:0.9, textTransform:'uppercase' }}>
      <span style={{ position:'absolute', left:size*0.04, top:size*0.04, color:ghost, opacity:0.35, mixBlendMode:'multiply' }}>Replicant</span>
      <span style={{ position:'relative', color }}>Replicant</span>
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// STAMP — animated "APPROVED"/"REJECTED"/etc.
// ─────────────────────────────────────────────────────────
function RPStamp({ children, variant='red', rotate=-6, size=22, style={} }) {
  const c = variant==='green' ? rpColors.stampGreen : variant==='blue' ? rpColors.stampBlue : rpColors.stampRed;
  return (
    <span className="stamp" style={{
      color:c, borderColor:c, fontSize:size, transform:`rotate(${rotate}deg)`, ...style,
    }}>{children}</span>
  );
}

// ─────────────────────────────────────────────────────────
// REDACTION — solid black censoring bar with optional real text underneath
// ─────────────────────────────────────────────────────────
function RPRedact({ children, width }) {
  return <span className="redacted" style={{ display:'inline-block', width }}>{children}</span>;
}

// ─────────────────────────────────────────────────────────
// MEMO HEADER — "DEPT. COMMUNIQUÉ" style bureaucratic bar
// ─────────────────────────────────────────────────────────
function RPMemoHeader({ title='INTERNAL MEMO', no='R-07-2186', classification='CLASSIFIED' }) {
  return (
    <div style={{ borderBottom:`2px solid ${rpColors.ink}`, paddingBottom:10, marginBottom:16, display:'flex', justifyContent:'space-between', alignItems:'flex-end', gap:12 }}>
      <div>
        <div className="t-eyebrow" style={{ color:rpColors.inkFaded, marginBottom:2 }}>DEPT. OF HUMAN AFFAIRS</div>
        <div className="t-display" style={{ fontSize:22, color:rpColors.ink }}>{title}</div>
      </div>
      <div style={{ textAlign:'right', fontFamily:'JetBrains Mono, monospace', fontSize:10, color:rpColors.inkFaded, lineHeight:1.5 }}>
        <div>FORM № {no}</div>
        <div style={{ color:rpColors.stampRed }}>■ {classification}</div>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// BUTTON — paper card with stamp-like action
// ─────────────────────────────────────────────────────────
function RPButton({ children, variant='primary', icon, full, onClick, style={} }) {
  const s = {
    primary:   { bg:rpColors.ink,       fg:rpColors.paper3, border:rpColors.ink },
    stamp:     { bg:rpColors.stampRed,  fg:'#F2EAD3',       border:rpColors.stampRed2 },
    ghost:     { bg:'transparent',      fg:rpColors.ink,    border:rpColors.ink },
    quiet:     { bg:rpColors.paper2,    fg:rpColors.ink,    border:rpColors.ink },
  }[variant];
  return (
    <button onClick={onClick} style={{
      background:s.bg, color:s.fg, border:`2px solid ${s.border}`,
      fontFamily:'Oswald, Impact, sans-serif', fontSize:15, fontWeight:600,
      letterSpacing:'0.08em', textTransform:'uppercase',
      padding:'13px 22px', cursor:'pointer',
      width: full?'100%':'auto',
      display:'inline-flex', alignItems:'center', justifyContent:'center', gap:10,
      boxShadow:'3px 3px 0 rgba(28,26,21,0.25)',
      transition:'transform .08s, box-shadow .08s',
      ...style,
    }}
    onMouseDown={e=>{e.currentTarget.style.transform='translate(2px,2px)'; e.currentTarget.style.boxShadow='1px 1px 0 rgba(28,26,21,0.25)'}}
    onMouseUp={e=>{e.currentTarget.style.transform=''; e.currentTarget.style.boxShadow='3px 3px 0 rgba(28,26,21,0.25)'}}
    onMouseLeave={e=>{e.currentTarget.style.transform=''; e.currentTarget.style.boxShadow='3px 3px 0 rgba(28,26,21,0.25)'}}
    >
      {icon && <span style={{ fontSize:16 }}>{icon}</span>}
      {children}
    </button>
  );
}

// ─────────────────────────────────────────────────────────
// PLAYER CHIP — the ID card for each player.
// ─────────────────────────────────────────────────────────
function RPPlayerChip({ name='SUBJECT 04', num='#04', status='ALIVE', portrait='H', accent=rpColors.ink, small=false }) {
  const h = small?56:78;
  return (
    <div style={{
      display:'flex', alignItems:'center', gap:10,
      background:rpColors.paper3, border:`1.5px solid ${rpColors.ink}`,
      padding: small?'6px 10px 6px 6px':'8px 14px 8px 8px',
      boxShadow:'2px 2px 0 rgba(28,26,21,0.3)',
      minHeight:h, position:'relative',
    }}>
      <div style={{
        width:small?44:62, height:small?44:62, flexShrink:0,
        background:accent, color:rpColors.paper3,
        display:'flex', alignItems:'center', justifyContent:'center',
        fontFamily:'Oswald, Impact, sans-serif', fontWeight:700,
        fontSize: small?20:28,
        border:`1.5px solid ${rpColors.ink}`,
        position:'relative', overflow:'hidden',
      }}>
        <span style={{ position:'relative', zIndex:1 }}>{portrait}</span>
        <div style={{
          position:'absolute', inset:0,
          background:'repeating-linear-gradient(45deg, transparent 0 3px, rgba(255,255,255,0.08) 3px 4px)',
        }}/>
      </div>
      <div style={{ flex:1, minWidth:0 }}>
        <div style={{ fontFamily:'JetBrains Mono, monospace', fontSize: small?9:10, color:rpColors.inkFaded, letterSpacing:1.4 }}>SUBJECT {num}</div>
        <div style={{ fontFamily:'Oswald, Impact, sans-serif', fontWeight:600, fontSize: small?15:18, color:rpColors.ink, textTransform:'uppercase', letterSpacing:'0.02em', whiteSpace:'nowrap', overflow:'hidden', textOverflow:'ellipsis' }}>{name}</div>
        <div style={{ display:'flex', alignItems:'center', gap:5, marginTop:2 }}>
          <span style={{ width:6, height:6, borderRadius:'50%', background: status==='ALIVE'? rpColors.stampGreen : status==='TERMINATED'? rpColors.stampRed : rpColors.inkFaded, display:'inline-block' }}/>
          <span className="t-eyebrow" style={{ fontSize:9, color:rpColors.inkFaded }}>{status}</span>
        </div>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// TIMER — typewriter countdown with ticker dots
// ─────────────────────────────────────────────────────────
function RPTimer({ value='02:14', label='TIME REMAINING', danger=false }) {
  const c = danger ? rpColors.stampRed : rpColors.ink;
  return (
    <div style={{ display:'inline-flex', flexDirection:'column', alignItems:'flex-start', gap:3 }}>
      <div className="t-eyebrow" style={{ color:rpColors.inkFaded, fontSize:10 }}>{label}</div>
      <div style={{ display:'flex', alignItems:'baseline', gap:8 }}>
        <span style={{ width:8, height:8, background:c, display:'inline-block' }} className="animate-blink"/>
        <span style={{ fontFamily:'JetBrains Mono, monospace', fontWeight:700, fontSize:38, color:c, letterSpacing:'0.04em', lineHeight:1 }}>{value}</span>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// TICKER — marquee strip, used in host chrome
// ─────────────────────────────────────────────────────────
function RPTicker({ items=['TRANSMISSION BEGINS','REPLICANT PROTOCOL R-07','CLASSIFIED'], bg=rpColors.ink, fg=rpColors.paper3 }) {
  const str = items.join('    ◼    ');
  return (
    <div style={{ background:bg, color:fg, overflow:'hidden', whiteSpace:'nowrap', padding:'6px 0', fontFamily:'JetBrains Mono, monospace', fontSize:11, letterSpacing:'0.2em' }}>
      <div style={{ display:'inline-block', animation:'tickerFeed 40s linear infinite', paddingLeft:'100%' }}>
        {str} ◼ {str} ◼ {str}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// PAPER SHEET — generic page surface w/ ruled lines + holes
// ─────────────────────────────────────────────────────────
function RPPaper({ children, style={}, withHoles=false, rotate=0, tone='paper' }) {
  const bg = tone==='pink' ? rpColors.paper2 : tone==='bright' ? rpColors.paper3 : rpColors.paper;
  return (
    <div className="paper-tex" style={{
      background:bg,
      boxShadow:'0 1px 0 rgba(0,0,0,0.04), 0 2px 6px rgba(60,50,30,0.18), 0 12px 24px rgba(60,50,30,0.08)',
      position:'relative', padding:24,
      transform: rotate?`rotate(${rotate}deg)`:'none',
      ...style,
    }}>
      {withHoles && (
        <div style={{ position:'absolute', left:8, top:20, bottom:20, display:'flex', flexDirection:'column', justifyContent:'space-around', alignItems:'center' }}>
          {[0,1,2].map(i => <div key={i} style={{ width:12, height:12, borderRadius:'50%', background:'rgba(0,0,0,0.18)', boxShadow:'inset 1px 1px 2px rgba(0,0,0,0.25)' }}/>)}
        </div>
      )}
      {children}
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// CHECKBOX / VOTE BOX — form-style
// ─────────────────────────────────────────────────────────
function RPCheckbox({ checked, mark='X', size=22, color=rpColors.stampRed }) {
  return (
    <span style={{
      display:'inline-flex', alignItems:'center', justifyContent:'center',
      width:size, height:size, border:`2px solid ${rpColors.ink}`, background:rpColors.paper3,
      fontFamily:'Special Elite, Courier, monospace', fontSize:size*0.9, lineHeight:1, color,
      fontWeight:700,
    }}>{checked?mark:''}</span>
  );
}

// ─────────────────────────────────────────────────────────
// TV CHROME — host-screen frame (dark broadcast)
// ─────────────────────────────────────────────────────────
function RPTVChrome({ children, title='DEPARTMENT BROADCAST', nodeId='NODE 04-7', phase='DAY 02' }) {
  return (
    <div className="broadcast-tex scanlines" style={{
      width:'100%', height:'100%', color:rpColors.paper3,
      display:'flex', flexDirection:'column',
      fontFamily:'Inter Tight, sans-serif',
      position:'relative', overflow:'hidden',
    }}>
      {/* header bar */}
      <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center', padding:'14px 28px', borderBottom:`1px solid ${rpColors.broadcastRule}`, flexShrink:0 }}>
        <div style={{ display:'flex', alignItems:'center', gap:14 }}>
          <span style={{ width:10, height:10, background:rpColors.stampRed, display:'inline-block' }} className="animate-blink"/>
          <span className="t-eyebrow" style={{ color:rpColors.stampRed, fontSize:11 }}>● REC · LIVE TRANSMISSION</span>
        </div>
        <div className="t-eyebrow" style={{ color:rpColors.paper3, fontSize:11 }}>{title}</div>
        <div style={{ display:'flex', gap:18, fontFamily:'JetBrains Mono, monospace', fontSize:11, color:rpColors.inkFaded }}>
          <span>{nodeId}</span><span style={{ color:rpColors.cyan }}>{phase}</span>
        </div>
      </div>
      {/* body */}
      <div style={{ flex:1, minHeight:0, position:'relative' }}>{children}</div>
      {/* footer ticker */}
      <div style={{ borderTop:`1px solid ${rpColors.broadcastRule}`, flexShrink:0 }}>
        <RPTicker bg={rpColors.broadcast2} fg={rpColors.cyan} items={['CITIZENS REMINDED TO COOPERATE','DETECTION MANDATORY','PROTOCOL R-07 ACTIVE','REPORT SUSPICIOUS BEHAVIOR','THE DEPARTMENT APPRECIATES YOUR COMPLIANCE']}/>
      </div>
      {/* corner registration marks */}
      {[['tl',8,8],['tr',8,8],['bl',8,8],['br',8,8]].map(([pos,x,y]) => {
        const s = { position:'absolute', width:16, height:16, border:`1px solid ${rpColors.cyan}`, opacity:0.5 };
        if (pos[0]==='t') s.top = y; else s.bottom = y;
        if (pos[1]==='l') s.left = x; else s.right = x;
        return <div key={pos} style={s}/>;
      })}
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// MOBILE FRAME — phone bezel
// ─────────────────────────────────────────────────────────
function RPPhone({ children, width=360, height=760 }) {
  return (
    <div style={{
      width:width+16, height:height+16, background:rpColors.ink, borderRadius:44,
      padding:8, boxShadow:'0 20px 50px rgba(0,0,0,0.2), inset 0 0 0 2px #2a2820',
    }}>
      <div style={{ width, height, background:rpColors.paper, borderRadius:36, overflow:'hidden', position:'relative' }}>
        <div style={{ position:'absolute', top:10, left:'50%', transform:'translateX(-50%)', width:110, height:28, background:rpColors.ink, borderRadius:20, zIndex:10 }}/>
        {children}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────
// MOBILE STATUS BAR
// ─────────────────────────────────────────────────────────
function RPMobileStatusBar() {
  return (
    <div style={{
      display:'flex', justifyContent:'space-between', alignItems:'center',
      padding:'14px 28px 4px', fontFamily:'JetBrains Mono, monospace',
      fontSize:12, color:rpColors.ink, fontWeight:700,
    }}>
      <span>21:47</span>
      <span style={{ display:'flex', gap:6, alignItems:'center' }}>
        <span>●●●</span>
        <span>100%</span>
      </span>
    </div>
  );
}

Object.assign(window, {
  rpColors, RPSeal, RPWordmark, RPStamp, RPRedact, RPMemoHeader,
  RPButton, RPPlayerChip, RPTimer, RPTicker, RPPaper, RPCheckbox,
  RPTVChrome, RPPhone, RPMobileStatusBar,
});
