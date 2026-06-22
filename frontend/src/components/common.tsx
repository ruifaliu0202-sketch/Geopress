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
      <Card sx={{ height: '100%' }}>
        <CardContent sx={{ height: '100%', minHeight: 132 }}>
          <Typography variant="body2" color="text.secondary">
            {label}
          </Typography>
          <Typography
            variant="h1"
            color={tone === 'error' ? 'error.main' : 'text.primary'}
            sx={{ mt: 1, overflowWrap: 'anywhere' }}
          >
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
    <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 2, overflow: 'hidden', minWidth: 0 }}>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        alignItems={{ xs: 'stretch', sm: 'center' }}
        justifyContent="space-between"
        spacing={1.5}
        sx={{ px: { xs: 1.5, sm: 2 }, py: 1.5, minWidth: 0 }}
      >
        <Typography variant="h3" sx={{ minWidth: 0, overflowWrap: 'anywhere' }}>
          {title}
        </Typography>
        {action && (
          <Box
            sx={{
              display: 'flex',
              justifyContent: { xs: 'flex-start', sm: 'flex-end' },
              maxWidth: '100%',
              minWidth: 0,
              '& > .MuiStack-root': {
                flexWrap: 'wrap',
              },
              '& .MuiButton-root': {
                whiteSpace: 'nowrap',
              },
            }}
          >
            {action}
          </Box>
        )}
      </Stack>
      <Divider />
      <Box sx={{ p: { xs: 1.5, sm: 2 }, overflowX: 'auto', WebkitOverflowScrolling: 'touch', minWidth: 0 }}>
        {children}
      </Box>
    </Paper>
  );
}

export function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <Stack
      direction={{ xs: 'column', sm: 'row' }}
      justifyContent="space-between"
      alignItems={{ xs: 'flex-start', sm: 'center' }}
      spacing={{ xs: 0.25, sm: 2 }}
      sx={{ py: 1.25, minWidth: 0 }}
    >
      <Typography color="text.secondary" sx={{ flexShrink: 0 }}>
        {label}
      </Typography>
      <Typography fontWeight={700} sx={{ textAlign: { xs: 'left', sm: 'right' }, overflowWrap: 'anywhere', minWidth: 0 }}>
        {value}
      </Typography>
    </Stack>
  );
}
