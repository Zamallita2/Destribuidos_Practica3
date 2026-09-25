import os

log_path = 'rejected_flights.log'
csv_path = 'dataset/vuelos_rechazados.csv'

if not os.path.exists(log_path):
    print('Log file not found.')
    exit(1)

with open(log_path, 'r', encoding='utf-8') as f_in, open(csv_path, 'w', encoding='utf-8') as f_out:
    f_out.write('Fecha,Hora,Origen,Destino,IDAvion,Estado,Puerta,Motivo Rechazo\n')
    for line in f_in:
        line = line.strip()
        if line.startswith('RECHAZADO:'):
            content = line.replace('RECHAZADO: ', '')
            parts = content.split(' | Motivo: ')
            if len(parts) == 2:
                flight_data = parts[0].strip()
                reason = parts[1].strip().replace('"', '""')
                f_out.write(f'{flight_data},"{reason}"\n')

print(f'CSV generado exitosamente en {csv_path}')
