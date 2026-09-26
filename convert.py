import csv
import os
import sys

base_dir = os.path.dirname(os.path.abspath(__file__))
log_path = os.path.join(base_dir, 'rejected_flights.log')
csv_path = os.path.join(base_dir, 'dataset', 'vuelos_rechazados.csv')

if not os.path.exists(log_path):
    print('Log file not found.')
    sys.exit(1)

with open(log_path, 'r', encoding='utf-8') as f_in, open(csv_path, 'w', encoding='utf-8', newline='') as f_out:
    writer = csv.writer(f_out)
    writer.writerow(['Fecha', 'Hora', 'Origen', 'Destino', 'IDAvion', 'Estado', 'Puerta', 'Motivo Rechazo'])
    for line in f_in:
        line = line.strip()
        if line.startswith('RECHAZADO:'):
            content = line.replace('RECHAZADO: ', '', 1)
            parts = content.split(' | Motivo: ')
            if len(parts) == 2:
                flight_data = [field.strip() for field in parts[0].split(',')]
                writer.writerow(flight_data + [parts[1].strip()])

print(f'CSV generado exitosamente en {csv_path}')
