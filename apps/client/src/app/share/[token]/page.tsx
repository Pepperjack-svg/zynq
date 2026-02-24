'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Loader2, Cloud, Download } from 'lucide-react';
import { formatBytes } from '@/lib/auth';
import { publicApi } from '@/lib/api';
import {
  getFileIcon,
  getIconColor,
  getIconBgColor,
} from '@/features/file/utils/file-icons';
import { cn } from '@/lib/utils';

interface SharedFile {
  id: string;
  name: string;
  mimeType: string;
  size: number;
  owner: string;
  ownerId: string;
  createdAt: string;
  isFolder: boolean;
  hasContent: boolean;
}

export default function PublicSharePage() {
  const { token } = useParams<{ token: string }>();
  const [file, setFile] = useState<SharedFile | null>(null);
  const [loading, setLoading] = useState(true);
  const [downloading, setDownloading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [error, setError] = useState('');

  const getPreviewType = (mimeType: string, name: string) => {
    if (mimeType.startsWith('image/')) return 'image' as const;
    if (mimeType.startsWith('video/')) return 'video' as const;
    if (mimeType.startsWith('audio/')) return 'audio' as const;
    if (mimeType === 'application/pdf') return 'pdf' as const;
    if (
      mimeType.startsWith('text/') ||
      ['txt', 'md', 'json', 'csv', 'xml', 'yaml', 'yml', 'log'].includes(
        name.split('.').pop()?.toLowerCase() || '',
      )
    ) {
      return 'text' as const;
    }
    return 'none' as const;
  };

  const previewType = file ? getPreviewType(file.mimeType, file.name) : 'none';

  const fetchFile = useCallback(async () => {
    try {
      const data = await publicApi.getShare(token);
      setFile(data);
    } catch {
      setError('This link is invalid or has expired.');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    if (!token) return;
    fetchFile();
  }, [token, fetchFile]);

  useEffect(() => {
    let stale = false;
    let url: string | null = null;

    const loadPreview = async () => {
      if (!file?.hasContent || previewType === 'none') return;
      setPreviewLoading(true);
      try {
        const { blob } = await publicApi.downloadShare(token);
        if (stale) return;

        if (previewType === 'text') {
          const text = await blob.text();
          if (!stale) setTextContent(text);
        } else {
          url = URL.createObjectURL(blob);
          if (!stale) setBlobUrl(url);
        }
      } catch {
        // Keep download available even if preview fails.
      } finally {
        if (!stale) setPreviewLoading(false);
      }
    };

    setBlobUrl(null);
    setTextContent(null);
    void loadPreview();

    return () => {
      stale = true;
      if (url) URL.revokeObjectURL(url);
    };
  }, [file?.id, file?.hasContent, previewType, token]);

  const handleDownload = async () => {
    if (!file?.hasContent) return;
    setDownloading(true);
    try {
      const { blob, fileName } = await publicApi.downloadShare(token);
      const blobUrl = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = blobUrl;
      a.download = fileName || file.name || 'download';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(blobUrl);
    } catch {
      setError('Download failed. Please try again.');
    } finally {
      setDownloading(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background to-primary/5">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center text-center p-6 bg-gradient-to-br from-background to-primary/5">
        <Cloud className="h-10 w-10 text-primary mb-3" />
        <h1 className="text-xl font-semibold">{error}</h1>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gradient-to-br from-background to-primary/5 p-4">
      <Card className="w-full max-w-4xl p-6 space-y-6 shadow-lg border-2">
        <div className="flex flex-col items-center space-y-3">
          <Cloud className="h-10 w-10 text-primary" />
          <h1 className="text-2xl font-bold text-center">zynqCloud Share</h1>
        </div>

        <div className="flex flex-col items-center space-y-2 text-center">
          {file &&
            (() => {
              const IconComponent = getFileIcon(
                file.name,
                file.mimeType,
                file.isFolder,
              );
              const iconColor = getIconColor(
                file.name,
                file.mimeType,
                file.isFolder,
              );
              const iconBgColor = getIconBgColor(
                file.name,
                file.mimeType,
                file.isFolder,
              );
              return (
                <div
                  className={cn(
                    'h-14 w-14 rounded-xl flex items-center justify-center',
                    iconBgColor,
                  )}
                >
                  <IconComponent className={cn('h-7 w-7', iconColor)} />
                </div>
              );
            })()}
          <p className="text-lg font-medium break-all max-w-full">
            {file?.name}
          </p>
          <p className="text-sm text-muted-foreground">
            {formatBytes(file?.size || 0)}
          </p>
          {file?.owner && (
            <p className="text-xs text-muted-foreground">
              Shared by {file.owner}
            </p>
          )}
        </div>

        {file?.hasContent && (
          <div className="w-full rounded-lg border bg-muted/20 min-h-[280px] max-h-[60vh] overflow-auto flex items-center justify-center">
            {previewLoading ? (
              <Loader2 className="h-8 w-8 animate-spin text-primary" />
            ) : previewType === 'image' && blobUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={blobUrl}
                alt={file.name}
                className="max-w-full max-h-[56vh] object-contain p-2"
              />
            ) : previewType === 'video' && blobUrl ? (
              <video
                src={blobUrl}
                controls
                className="max-w-full max-h-[56vh] p-2"
              />
            ) : previewType === 'audio' && blobUrl ? (
              <div className="w-full p-6">
                <audio src={blobUrl} controls className="w-full" />
              </div>
            ) : previewType === 'pdf' && blobUrl ? (
              <embed
                src={blobUrl}
                type="application/pdf"
                className="w-full min-h-[56vh]"
              />
            ) : previewType === 'text' && textContent !== null ? (
              <pre className="w-full p-4 text-xs whitespace-pre-wrap break-words">
                {textContent}
              </pre>
            ) : (
              <p className="text-sm text-muted-foreground py-10">
                Preview not available for this file type.
              </p>
            )}
          </div>
        )}

        <Button
          size="lg"
          className="w-full"
          disabled={downloading}
          onClick={handleDownload}
        >
          {downloading ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Download className="mr-2 h-4 w-4" />
          )}
          {downloading ? 'Downloading...' : 'Download File'}
        </Button>

        <p className="text-xs text-muted-foreground mt-3">
          Shared securely via <span className="font-semibold">zynqCloud</span>
        </p>
      </Card>
    </div>
  );
}
