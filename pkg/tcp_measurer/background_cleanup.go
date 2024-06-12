package tcpmeasurer

import "time"

func (s *Service) CleanOld() {
	ticker := time.NewTicker(s.cleanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.CleanIt()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) CleanIt() {
	dropBefore := time.Now().Add(-1 * time.Hour)
	s.dataMUSeq.Lock()
	defer s.dataMUSeq.Unlock()
	for remoteHost := range s.dataSeq {
		for key := range s.dataSeq[remoteHost] {
			if s.dataSeq[remoteHost][key].EventTime.Before(dropBefore) {
				delete(s.dataSeq[remoteHost], key)
			}
		}
	}
}
