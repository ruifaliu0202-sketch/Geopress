import type { ReactNode } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Typography,
} from '@mui/material';

export type DialogBaseProps = {
  open: boolean;
  token: string;
  workspaceId: string;
  onClose: () => void;
  onCreated: () => void;
};

export function FormDialog({
  title,
  open,
  error,
  submitting,
  children,
  onClose,
  onSubmit,
}: {
  title: string;
  open: boolean;
  error: string | null;
  submitting: boolean;
  children: ReactNode;
  onClose: () => void;
  onSubmit: () => void;
}) {
  return (
    <Dialog open={open} onClose={submitting ? undefined : onClose} fullWidth maxWidth="sm">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          {children}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          取消
        </Button>
        <Button onClick={onSubmit} disabled={submitting} variant="contained">
          确认
        </Button>
      </DialogActions>
    </Dialog>
  );
}

export function SelectField({
  label,
  value,
  items,
  onChange,
}: {
  label: string;
  value: string;
  items: Array<{ value: string; label: string }>;
  onChange: (value: string) => void;
}) {
  return (
    <FormControl fullWidth disabled={items.length === 0}>
      <InputLabel>{label}</InputLabel>
      <Select label={label} value={value} onChange={(event) => onChange(String(event.target.value))}>
        {items.map((item) => (
          <MenuItem key={item.value} value={item.value}>
            {item.label}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}

export function MetricCard({
  label,
  value,
  helper,
  tone = 'primary',
}: {
  label: string;
  value: number;
  helper: string;
  tone?: 'primary' | 'error';
}) {
  return (
    <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
      <Card>
        <CardContent>
          <Typography variant="body2" color="text.secondary">
            {label}
          </Typography>
          <Typography variant="h1" color={tone === 'error' ? 'error.main' : 'text.primary'} sx={{ mt: 1 }}>
            {value}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {helper}
          </Typography>
        </CardContent>
      </Card>
    </Grid>
  );
}

export function Section({ title, action, children }: { title: string; action?: ReactNode; children: ReactNode }) {
  return (
    <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 2, overflow: 'hidden' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={2} sx={{ px: 2, py: 1.5 }}>
        <Typography variant="h3">{title}</Typography>
        {action}
      </Stack>
      <Divider />
      <Box sx={{ p: 2, overflowX: 'auto' }}>{children}</Box>
    </Paper>
  );
}

export function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <Stack direction="row" justifyContent="space-between" spacing={2} sx={{ py: 1.25 }}>
      <Typography color="text.secondary">{label}</Typography>
      <Typography fontWeight={700} sx={{ textAlign: 'right', overflowWrap: 'anywhere' }}>
        {value}
      </Typography>
    </Stack>
  );
}
