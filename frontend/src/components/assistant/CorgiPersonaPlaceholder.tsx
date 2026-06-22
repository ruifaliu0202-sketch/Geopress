import { Box } from '@mui/material';

export function CorgiPersonaPlaceholder() {
  return (
    <Box
      aria-hidden="true"
      sx={{
        width: 68,
        height: 68,
        display: 'grid',
        placeItems: 'center',
        position: 'relative',
      }}
    >
      <Box
        sx={{
          width: 50,
          height: 44,
          borderRadius: '50% 50% 45% 45%',
          bgcolor: '#d88a3d',
          border: '2px solid #7a4b24',
          position: 'relative',
          boxShadow: 'inset 0 -7px 0 rgba(255,255,255,0.45)',
          '&::before, &::after': {
            content: '""',
            position: 'absolute',
            top: -14,
            width: 18,
            height: 26,
            bgcolor: '#d88a3d',
            border: '2px solid #7a4b24',
            borderBottom: 0,
            transformOrigin: 'bottom center',
          },
          '&::before': {
            left: 2,
            borderRadius: '80% 20% 20% 20%',
            transform: 'rotate(-24deg)',
          },
          '&::after': {
            right: 2,
            borderRadius: '20% 80% 20% 20%',
            transform: 'rotate(24deg)',
          },
        }}
      >
        <Box
          sx={{
            position: 'absolute',
            left: 10,
            top: 16,
            width: 6,
            height: 6,
            borderRadius: '50%',
            bgcolor: '#1f2933',
            boxShadow: '24px 0 0 #1f2933',
          }}
        />
        <Box
          sx={{
            position: 'absolute',
            left: '50%',
            bottom: 9,
            width: 20,
            height: 14,
            transform: 'translateX(-50%)',
            borderRadius: '50% 50% 45% 45%',
            bgcolor: '#fff7ed',
            border: '1px solid rgba(122,75,36,0.35)',
            '&::before': {
              content: '""',
              position: 'absolute',
              left: '50%',
              top: 3,
              width: 7,
              height: 5,
              transform: 'translateX(-50%)',
              borderRadius: '50%',
              bgcolor: '#1f2933',
            },
          }}
        />
      </Box>
    </Box>
  );
}
