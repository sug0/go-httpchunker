import os
import sys

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(f'Usage: python {sys.argv[0]} <input dir> <destiny file>')
        sys.exit(1)
    inputdir = sys.argv[1]
    destfile = sys.argv[2]
    with open(destfile, 'wb') as dest:
        for filename in sorted(os.listdir(inputdir)):
            filename = f'{inputdir}/{filename}'
            with open(filename, 'rb') as f:
                dest.write(f.read())
            os.remove(filename)
    os.rmdir(inputdir)
