'use client';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Copy } from 'lucide-react';
import { toast } from '@/hooks/use-toast';

interface PublicLinkDialogProps {
  publicLink: string | null;
  fileName?: string | null;
  onClose: () => void;
}

export function PublicLinkDialog({
  publicLink,
  fileName,
  onClose,
}: PublicLinkDialogProps) {
  const handleCopy = async () => {
    if (publicLink) {
      await navigator.clipboard.writeText(publicLink);
      toast({ title: 'Copied to clipboard!' });
    }
  };

  return (
    <Dialog open={!!publicLink} onOpenChange={() => onClose()}>
      <DialogContent className="w-[95vw] max-w-xl">
        <DialogHeader>
          <DialogTitle>Public Share Link</DialogTitle>
          <DialogDescription>
            Copy and share this link with others.
          </DialogDescription>
        </DialogHeader>
        {fileName ? (
          <p className="text-sm text-muted-foreground break-all rounded-md bg-muted/40 px-3 py-2">
            {fileName}
          </p>
        ) : null}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 bg-muted px-3 py-2 rounded-md">
          <Input
            value={publicLink || ''}
            readOnly
            className="text-sm font-mono break-all"
          />
          <Button
            size="icon"
            variant="secondary"
            onClick={handleCopy}
            className="self-end sm:self-auto"
          >
            <Copy className="h-4 w-4" />
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
