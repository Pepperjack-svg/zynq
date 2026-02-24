'use client';

import { useParams } from 'next/navigation';
import { PublicShareView } from '@/features/share/components/public-share-view';

export default function PublicSharePage() {
  const { token } = useParams<{ token: string }>();
  if (!token) return null;
  return <PublicShareView token={token} />;
}
