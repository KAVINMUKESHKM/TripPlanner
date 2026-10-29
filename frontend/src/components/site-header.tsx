"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { logout, currentUser } from "@/lib/api";
import type { User } from "@/lib/types";
import Button from "@mui/material/Button";

export function SiteHeader() {
  const [user, setUser] = useState<User | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => { currentUser().then(setUser).catch(() => setUser(null)); }, []);

  async function signOut() {
    setBusy(true);
    try { await logout(); } finally { setUser(null); setBusy(false); }
  }

  return (
    <header className="site-header">
      <Link className="brand" href="/" aria-label="Trip Planner home"><span className="brand-mark">T</span><span>Trip Planner</span></Link>
      <nav className="main-nav" aria-label="Main navigation">
        <Link href="/plan">Plan a trip</Link>
        {user ? <Link href="/trips">My trips</Link> : null}
      </nav>
      <div className="header-account">
        {user ? <><span className="account-name">{user.full_name || user.email}</span><Button size="small" disabled={busy} onClick={signOut}>{busy ? "Signing out…" : "Sign out"}</Button></> : <><Button component={Link} size="small" href="/login" color="inherit">Sign in</Button><Button component={Link} size="small" href="/signup" variant="contained">Create account</Button></>}
      </div>
    </header>
  );
}
